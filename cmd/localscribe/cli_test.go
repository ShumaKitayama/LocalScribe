package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

type fakeCLIService struct {
	settings      domain.AppSettings
	job           domain.TranscriptionJob
	getJobCalls   int
	startErr      error
	progressFirst bool
	terminalState domain.JobState
}

func (f *fakeCLIService) StartTranscription(req domain.TranscriptionRequest) (string, error) {
	if f.startErr != nil {
		return "", f.startErr
	}
	state := domain.StateQueued
	if f.terminalState != "" {
		state = f.terminalState
	}
	f.job = domain.TranscriptionJob{
		ID:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		State:     state,
		Request:   req,
		ResultDir: "/tmp/out/lecture_20261027_001",
		Progress: domain.JobProgress{
			JobID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			State:   state,
			Message: "待機しています",
		},
	}
	return f.job.ID, nil
}

func (f *fakeCLIService) GetJob(string) (domain.TranscriptionJob, error) {
	if f.terminalState != "" {
		job := f.job
		job.State = f.terminalState
		job.Progress.State = f.terminalState
		return job, nil
	}
	f.getJobCalls++
	if f.progressFirst && f.getJobCalls == 1 {
		job := f.job
		job.State = domain.StateTranscribing
		job.Progress.State = domain.StateTranscribing
		job.Progress.Message = "音声を認識しています"
		return job, nil
	}
	job := f.job
	job.State = domain.StateCompleted
	job.Progress.State = domain.StateCompleted
	job.Progress.Message = "完了しました"
	return job, nil
}

func (f *fakeCLIService) CancelJob(string) error { return nil }

func (f *fakeCLIService) ListModels() ([]domain.ModelInfo, error) {
	return []domain.ModelInfo{{ID: domain.ModelLargeV3, Status: domain.ModelMissing}}, nil
}

func (f *fakeCLIService) InstallModel(context.Context, string) error { return nil }

func (f *fakeCLIService) GetSettings() (domain.AppSettings, error) {
	if f.settings.OutputDir != "" {
		return f.settings, nil
	}
	return domain.AppSettings{
		SchemaVersion:  domain.SchemaVersion,
		DefaultModelID: domain.ModelLargeV3,
		Language:       domain.LanguageJA,
		OutputDir:      "/tmp/transcripts",
		ExportFormats:  []domain.ExportFormat{domain.FormatJSON},
	}, nil
}

func (f *fakeCLIService) Close() {}

func testBackends(svc AppService) commandBackends {
	return commandBackends{
		transcribe: func() (AppService, func(), error) {
			return svc, func() {}, nil
		},
		models: func() (AppService, error) {
			return svc, nil
		},
	}
}

func TestTranscribeMissingFlagsExitUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"localscribe", "transcribe", "file.m4a"}, &stdout, &stderr, testBackends(&fakeCLIService{}), t.TempDir())
	if code != exitUsage {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty: %s", stdout.String())
	}
}

func TestTranscribeSuccessJSONOnStdoutOnly(t *testing.T) {
	fake := &fakeCLIService{progressFirst: true}
	var stdout, stderr bytes.Buffer
	args := []string{
		"localscribe", "transcribe", "/tmp/a.m4a",
		"--model", "large-v3",
		"--language", "ja",
		"--format", "txt,json",
	}
	code := Run(args, &stdout, &stderr, testBackends(fake), t.TempDir())
	if code != exitOK {
		t.Fatalf("code %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "音声を認識しています") {
		t.Fatalf("progress missing on stderr: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), "音声を認識しています") {
		t.Fatal("progress leaked to stdout")
	}
	var out transcribeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("stdout json: %v body=%s", err, stdout.String())
	}
	if out.State != "completed" || out.OutputDir == "" || len(out.Files) == 0 {
		t.Fatalf("output %+v", out)
	}
}

func TestTranscribeCancelledExitCode(t *testing.T) {
	fake := &fakeCLIService{terminalState: domain.StateCancelled}
	var stdout, stderr bytes.Buffer
	args := []string{
		"localscribe", "transcribe", "/tmp/a.m4a",
		"--model", "large-v3",
		"--language", "ja",
		"--format", "json",
	}
	code := Run(args, &stdout, &stderr, testBackends(fake), t.TempDir())
	if code != exitCanceled {
		t.Fatalf("code %d", code)
	}
}

func TestModelsListWritesJSONToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"localscribe", "models", "list"}, &stdout, &stderr, testBackends(&fakeCLIService{}), t.TempDir())
	if code != exitOK {
		t.Fatal(code)
	}
	var models []domain.ModelInfo
	if err := json.Unmarshal(stdout.Bytes(), &models); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr %s", stderr.String())
	}
}

func TestDoctorRunsWithoutPanic(t *testing.T) {
	var stdout, stderr bytes.Buffer
	home := t.TempDir()
	code := Run([]string{"localscribe", "doctor"}, &stdout, &stderr, testBackends(&fakeCLIService{}), home)
	if code != exitOK && code != exitRuntime {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stdout.String(), "LocalScribe doctor") {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestTranscribeCompletesWithoutBusyLoop(t *testing.T) {
	fake := &fakeCLIService{}
	start := time.Now()
	var stdout, stderr bytes.Buffer
	args := []string{
		"localscribe", "transcribe", "/tmp/a.m4a",
		"--model", "large-v3",
		"--language", "ja",
		"--format", "json",
	}
	code := Run(args, &stdout, &stderr, testBackends(fake), t.TempDir())
	if code != exitOK {
		t.Fatalf("code %d", code)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("transcribe took too long")
	}
}

func TestUsageDoesNotLoadTranscriptionService(t *testing.T) {
	backends := commandBackends{
		transcribe: func() (AppService, func(), error) {
			t.Fatal("文字起こしサービスを起動した")
			return nil, nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"localscribe", "transcribe", "file.m4a"}, &stdout, &stderr, backends, t.TempDir())
	if code != exitUsage {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
}

func TestDoctorDoesNotLoadService(t *testing.T) {
	backends := commandBackends{
		transcribe: func() (AppService, func(), error) {
			t.Fatal("文字起こしサービスを起動した")
			return nil, nil, nil
		},
		models: func() (AppService, error) {
			t.Fatal("モデルサービスを起動した")
			return nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"localscribe", "doctor"}, &stdout, &stderr, backends, t.TempDir())
	if code != exitOK && code != exitRuntime {
		t.Fatalf("code %d stdout %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "LocalScribe doctor") {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestUnknownModelDoesNotLoadService(t *testing.T) {
	backends := commandBackends{
		models: func() (AppService, error) {
			t.Fatal("モデルサービスを起動した")
			return nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"localscribe", "models", "install", "nope"}, &stdout, &stderr, backends, t.TempDir())
	if code != exitUsage {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
}
