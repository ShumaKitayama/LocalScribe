package application

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/audio"
	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/engine"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
	"github.com/ShumaKitayama/localscribe/internal/storage"
)

type fakeProber struct {
	info audio.ProbeInfo
}

func (f fakeProber) Probe(context.Context, string) (audio.ProbeInfo, error) {
	return f.info, nil
}

type fakeConverter struct {
	workDir string
}

func (f fakeConverter) Convert(_ context.Context, workDir, _ string, outputWAV string, _ audio.ProbeInfo) error {
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(outputWAV, []byte("wav"), 0o644)
}

type fakeEngine struct {
	mu              sync.Mutex
	hold            chan struct{}
	maxConcurrent   int
	peak            int
	failNext        bool
	segments        []domain.Segment
	transcribeCalls int
}

func (f *fakeEngine) Transcribe(ctx context.Context, _ engine.Request, _ func(engine.Progress)) ([]domain.Segment, error) {
	f.mu.Lock()
	f.transcribeCalls++
	f.peak++
	if f.peak > f.maxConcurrent {
		f.maxConcurrent = f.peak
	}
	f.mu.Unlock()

	if f.failNext {
		f.mu.Lock()
		f.failNext = false
		f.mu.Unlock()
		return nil, &domain.UserError{Message: "認識に失敗しました", Detail: "fake failure"}
	}

	if f.hold != nil {
		select {
		case <-ctx.Done():
			return nil, engine.ErrCancelled
		case <-f.hold:
		}
	}
	f.mu.Lock()
	f.peak--
	f.mu.Unlock()
	if f.segments != nil {
		return f.segments, nil
	}
	return []domain.Segment{{StartMS: 0, EndMS: 1000, Text: "hello"}}, nil
}

var testJobIDs = []string{
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	"cccccccccccccccccccccccccccccccc",
	"dddddddddddddddddddddddddddddddd",
}

func setupService(t *testing.T, eng *fakeEngine) (*Service, string, string, string) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	jobsDir := filepath.Join(root, "jobs")
	outputDir := filepath.Join(root, "output")
	workDir := macos.WorkCacheDir(home)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installFakeModel(t, home, domain.ModelLargeV3)

	source := filepath.Join(root, "lecture.m4a")
	if err := os.WriteFile(source, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}

	jobSeq := 0
	var jobSeqMu sync.Mutex
	svc, err := NewService(ServiceConfig{
		Home:      home,
		JobsDir:   jobsDir,
		Prober:    fakeProber{info: audio.ProbeInfo{DurationMS: 1000, SampleRate: 16000, Channels: 1}},
		Converter: fakeConverter{},
		Engine:    eng,
		IDGen: func() (string, error) {
			jobSeqMu.Lock()
			defer jobSeqMu.Unlock()
			if jobSeq >= len(testJobIDs) {
				return "", &domain.UserError{Message: "テスト用IDを使い切りました", Detail: "test ids"}
			}
			id := testJobIDs[jobSeq]
			jobSeq++
			return id, nil
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, source, outputDir, home
}

func installFakeModel(t *testing.T, home, modelID string) {
	fileName, ok := domain.ModelFileName(modelID)
	if !ok {
		t.Fatal("model file name")
	}
	dir := macos.ModelsDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fileName)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	minBytes, _ := domain.ModelMinBytes(modelID)
	if err := file.Truncate(minBytes); err != nil {
		t.Fatal(err)
	}
	file.Close()
}

func sampleRequest(source, outputDir string) domain.TranscriptionRequest {
	return domain.TranscriptionRequest{
		SourcePath:    source,
		ModelID:       domain.ModelLargeV3,
		Language:      domain.LanguageJA,
		OutputDir:     outputDir,
		ExportFormats: []domain.ExportFormat{domain.FormatJSON, domain.FormatTXT},
	}
}

func waitForJobState(t *testing.T, svc *Service, jobID string, state domain.JobState) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, err := svc.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob: %v", err)
		}
		if job.State == state {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach %s", jobID, state)
}

func TestTwoJobsDoNotTranscribeConcurrently(t *testing.T) {
	hold := make(chan struct{})
	eng := &fakeEngine{hold: hold, segments: []domain.Segment{{StartMS: 0, EndMS: 1, Text: "a"}}}
	svc, source, outputDir, _ := setupService(t, eng)
	defer svc.Close()

	id1, err := svc.StartTranscription(sampleRequest(source, outputDir))
	if err != nil {
		t.Fatal(err)
	}
	id2, err := svc.StartTranscription(sampleRequest(source, outputDir))
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		j1, _ := svc.GetJob(id1)
		j2, _ := svc.GetJob(id2)
		transcribing := 0
		if j1.State == domain.StateTranscribing {
			transcribing++
		}
		if j2.State == domain.StateTranscribing {
			transcribing++
		}
		if transcribing > 1 {
			t.Fatalf("both jobs transcribing: %s %s", j1.State, j2.State)
		}
		if j1.State == domain.StateCompleted && j2.State == domain.StateCompleted {
			break
		}
		if j1.State == domain.StateTranscribing || j2.State == domain.StateTranscribing {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	close(hold)
	waitForJobState(t, svc, id1, domain.StateCompleted)
	waitForJobState(t, svc, id2, domain.StateCompleted)

	if eng.maxConcurrent > 1 {
		t.Fatalf("max concurrent transcribe %d", eng.maxConcurrent)
	}
}

func TestFailureDoesNotBlockNextJob(t *testing.T) {
	eng := &fakeEngine{failNext: true}
	svc, source, outputDir, _ := setupService(t, eng)
	defer svc.Close()

	id1, err := svc.StartTranscription(sampleRequest(source, outputDir))
	if err != nil {
		t.Fatal(err)
	}
	waitForJobState(t, svc, id1, domain.StateFailed)

	id2, err := svc.StartTranscription(sampleRequest(source, outputDir))
	if err != nil {
		t.Fatal(err)
	}
	waitForJobState(t, svc, id2, domain.StateCompleted)
}

func TestCancelRemovesWorkFiles(t *testing.T) {
	hold := make(chan struct{})
	eng := &fakeEngine{hold: hold}
	svc, source, outputDir, home := setupService(t, eng)
	defer svc.Close()

	id, err := svc.StartTranscription(sampleRequest(source, outputDir))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, _ := svc.GetJob(id)
		if job.State == domain.StateTranscribing {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := svc.CancelJob(id); err != nil {
		t.Fatal(err)
	}
	waitForJobState(t, svc, id, domain.StateCancelled)

	workWAV := filepath.Join(macos.WorkCacheDir(home), id+".wav")
	if _, err := os.Stat(workWAV); !os.IsNotExist(err) {
		t.Fatalf("work wav still exists: %v", err)
	}
}

func TestInterruptedJobsAreNotAutoStarted(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	jobsDir := filepath.Join(root, "jobs")
	installFakeModel(t, home, domain.ModelLargeV3)

	store, err := storage.NewJobStoreDir(jobsDir)
	if err != nil {
		t.Fatal(err)
	}
	req := sampleRequest("/tmp/in.m4a", filepath.Join(root, "out"))
	job, err := domain.NewTranscriptionJob("dddddddddddddddddddddddddddddddd", req)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Transition(domain.StateTranscribing); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}

	eng := &fakeEngine{}
	svc, err := NewService(ServiceConfig{
		Home:      home,
		JobsDir:   jobsDir,
		Prober:    fakeProber{info: audio.ProbeInfo{DurationMS: 1, SampleRate: 16000, Channels: 1}},
		Converter: fakeConverter{},
		Engine:    eng,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	got, err := svc.GetJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateInterrupted {
		t.Fatalf("state %s", got.State)
	}
	time.Sleep(100 * time.Millisecond)
	if eng.transcribeCalls != 0 {
		t.Fatalf("engine calls %d", eng.transcribeCalls)
	}
}
