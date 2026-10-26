package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

func TestPrepareOutputDirDoesNotOverwrite(t *testing.T) {
	base := t.TempDir()
	source := "/media/lecture.m4a"
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

	first, err := PrepareOutputDir(base, source, now)
	if err != nil {
		t.Fatalf("first PrepareOutputDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(first, "marker"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}

	second, err := PrepareOutputDir(base, source, now)
	if err != nil {
		t.Fatalf("second PrepareOutputDir: %v", err)
	}
	if first == second {
		t.Fatalf("expected different dirs, both %q", first)
	}
	wantSecond := filepath.Join(base, "lecture_20261027_002")
	if second != wantSecond {
		t.Fatalf("second dir %q want %q", second, wantSecond)
	}
}

func TestSaveTranscriptAtomic(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	transcript := domain.Transcript{
		SchemaVersion: domain.SchemaVersion,
		ID:            "dddddddddddddddddddddddddddddddd",
		JobID:         "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		ModelID:       domain.ModelLargeV3,
		Language:      domain.LanguageJA,
		SourcePath:    "/tmp/lecture.m4a",
		DurationMS:    1000,
		CreatedAt:     now,
		Segments: []domain.Segment{
			{StartMS: 0, EndMS: 1000, Text: "hello"},
		},
	}
	if err := SaveTranscript(dir, transcript); err != nil {
		t.Fatalf("SaveTranscript: %v", err)
	}
	path := filepath.Join(dir, "transcript.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("transcript.json missing: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp transcript should not remain")
	}
	loaded, err := LoadTranscript(dir)
	if err != nil {
		t.Fatalf("LoadTranscript: %v", err)
	}
	if loaded.ID != transcript.ID || loaded.Segments[0].Text != "hello" {
		t.Fatalf("loaded mismatch: %+v", loaded)
	}
}
