package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicCreatesFinalFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "data.json")

	if err := WriteFileAtomic(target, []byte("ok")); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("final file missing: %v", err)
	}
	if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file should be removed")
	}
}

func TestWriteFileAtomicRenameFailureLeavesNoFinalOrTemp(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "data.json")
	orig := renameFile
	renameFile = func(string, string) error {
		return errors.New("rename failed")
	}
	t.Cleanup(func() { renameFile = orig })

	err := WriteFileAtomic(target, []byte("partial"))
	if err == nil {
		t.Fatal("expected rename error")
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("final file should not exist: %v", statErr)
	}
	if _, statErr := os.Stat(target + ".tmp"); !os.IsNotExist(statErr) {
		t.Fatalf("temp file should be removed: %v", statErr)
	}
}
