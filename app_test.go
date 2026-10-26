package main

import (
	"os"
	"path/filepath"
	"testing"

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
