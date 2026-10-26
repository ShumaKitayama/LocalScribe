package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/application"
	"github.com/ShumaKitayama/localscribe/internal/audio"
	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/engine"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
)

func TestSaveRecordingWritesWebM(t *testing.T) {
	home := t.TempDir()
	app := &App{home: home}
	data := []byte("fake-webm-bytes")
	result, err := app.SaveRecording(data)
	if err != nil {
		t.Fatal(err)
	}
	if result.SizeBytes != int64(len(data)) {
		t.Fatalf("size %d", result.SizeBytes)
	}
	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("empty file")
	}
	wantDir := filepath.Join(macos.WorkCacheDir(home), "recordings")
	if filepath.Dir(result.Path) != wantDir {
		t.Fatalf("dir %s want %s", filepath.Dir(result.Path), wantDir)
	}
}

func TestUpdateTranscriptPreservesRecognitionText(t *testing.T) {
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
	modelDir := macos.ModelsDir(home)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(modelDir, "ggml-large-v3.bin")
	file, err := os.Create(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	minBytes, _ := domain.ModelMinBytes(domain.ModelLargeV3)
	if err := file.Truncate(minBytes); err != nil {
		t.Fatal(err)
	}
	file.Close()

	source := filepath.Join(root, "lecture.m4a")
	if err := os.WriteFile(source, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}

	jobID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	transcriptID := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	segments := []domain.Segment{{StartMS: 0, EndMS: 500, Text: "認識原文"}}
	idSeq := 0
	fixedIDs := []string{jobID, transcriptID}

	svc, err := application.NewService(application.ServiceConfig{
		Home:      home,
		JobsDir:   jobsDir,
		Prober:    fakeProber{},
		Converter: fakeConverter{},
		Engine:    &fakeEngine{segments: segments},
		IDGen: func() (string, error) {
			if idSeq >= len(fixedIDs) {
				return "", &domain.UserError{Message: "テスト用IDを使い切りました", Detail: "test ids"}
			}
			id := fixedIDs[idSeq]
			idSeq++
			return id, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	app := &App{
		home:     home,
		svc:      svc,
		watch:    make(map[string]struct{}),
		lastEmit: make(map[string]string),
	}
	id, err := app.StartTranscription(domain.TranscriptionRequest{
		SourcePath:    source,
		SourceKind:    domain.SourceFile,
		ModelID:       domain.ModelLargeV3,
		Language:      domain.LanguageJA,
		OutputDir:     outputDir,
		ExportFormats: []domain.ExportFormat{domain.FormatTXT, domain.FormatJSON},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != jobID {
		t.Fatalf("job id %s", id)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, err := app.GetJob(jobID)
		if err != nil {
			t.Fatal(err)
		}
		if job.State == domain.StateCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	job, err := app.GetJob(jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.State != domain.StateCompleted {
		t.Fatalf("state %s", job.State)
	}
	if err := app.UpdateTranscript(job.TranscriptID, []domain.SegmentEdit{
		{Index: 0, EditedText: "編集後"},
	}); err != nil {
		t.Fatal(err)
	}
	transcript, err := app.GetTranscript(job.TranscriptID)
	if err != nil {
		t.Fatal(err)
	}
	if transcript.Segments[0].Text != "認識原文" {
		t.Fatalf("text %q", transcript.Segments[0].Text)
	}
	if transcript.Segments[0].EditedText == nil || *transcript.Segments[0].EditedText != "編集後" {
		t.Fatalf("edited %v", transcript.Segments[0].EditedText)
	}
}

type fakeProber struct{}

func (fakeProber) Probe(context.Context, string) (audio.ProbeInfo, error) {
	return audio.ProbeInfo{DurationMS: 1000, SampleRate: 16000, Channels: 1}, nil
}

type fakeConverter struct{}

func (fakeConverter) Convert(_ context.Context, workDir, _ string, outputWAV string, _ audio.ProbeInfo) error {
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(outputWAV, []byte("wav"), 0o644)
}

type fakeEngine struct {
	segments []domain.Segment
}

func (f *fakeEngine) Transcribe(context.Context, engine.Request, func(engine.Progress)) ([]domain.Segment, error) {
	return f.segments, nil
}
