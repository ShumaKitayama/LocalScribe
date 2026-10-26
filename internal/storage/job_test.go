package storage

import (
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

func sampleRequest(outputDir string) domain.TranscriptionRequest {
	return domain.TranscriptionRequest{
		SourcePath:    "/tmp/lecture.m4a",
		ModelID:       domain.ModelLargeV3,
		Language:      domain.LanguageJA,
		OutputDir:     outputDir,
		ExportFormats: []domain.ExportFormat{domain.FormatJSON},
	}
}

func TestMarkInterruptedOnStartup(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJobStoreDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

	queued, err := domain.NewTranscriptionJob("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", sampleRequest(output))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(queued); err != nil {
		t.Fatal(err)
	}

	transcribing, err := domain.NewTranscriptionJob("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", sampleRequest(output))
	if err != nil {
		t.Fatal(err)
	}
	if err := transcribing.Transition(domain.StateTranscribing); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(transcribing); err != nil {
		t.Fatal(err)
	}

	completed, err := domain.NewTranscriptionJob("cccccccccccccccccccccccccccccccc", sampleRequest(output))
	if err != nil {
		t.Fatal(err)
	}
	if err := completed.Transition(domain.StateTranscribing); err != nil {
		t.Fatal(err)
	}
	if err := completed.Transition(domain.StateExporting); err != nil {
		t.Fatal(err)
	}
	if err := completed.Transition(domain.StateCompleted); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(completed); err != nil {
		t.Fatal(err)
	}

	if err := store.MarkInterruptedOnStartup(now); err != nil {
		t.Fatalf("MarkInterruptedOnStartup: %v", err)
	}

	gotQueued, err := store.Load("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if gotQueued.State != domain.StateInterrupted {
		t.Fatalf("queued job state %q", gotQueued.State)
	}

	gotTranscribing, err := store.Load("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err != nil {
		t.Fatal(err)
	}
	if gotTranscribing.State != domain.StateInterrupted {
		t.Fatalf("transcribing job state %q", gotTranscribing.State)
	}

	gotCompleted, err := store.Load("cccccccccccccccccccccccccccccccc")
	if err != nil {
		t.Fatal(err)
	}
	if gotCompleted.State != domain.StateCompleted {
		t.Fatalf("completed job state %q", gotCompleted.State)
	}
}
