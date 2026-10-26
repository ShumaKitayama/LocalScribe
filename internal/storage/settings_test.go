package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
)

func TestSettingsSaveExpandsTilde(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := NewSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	settings, err := DefaultSettings(home)
	if err != nil {
		t.Fatal(err)
	}
	settings.OutputDir = "~/Documents/LocalScribe/Transcripts"
	if err := store.Save(settings); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	var saved domain.AppSettings
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	want := macos.DefaultTranscriptsDir(home)
	if saved.OutputDir != want {
		t.Fatalf("output_dir %q want %q", saved.OutputDir, want)
	}
}
