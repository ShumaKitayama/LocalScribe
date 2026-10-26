package macos_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
)

func TestBundledBinDirFromExecutable(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "LocalScribe.app", "Contents", "MacOS", "LocalScribe")
	dir, ok := macos.BundledBinDirFromExecutable(exe)
	if !ok {
		t.Fatal("expected bundled dir")
	}
	want := filepath.Join(root, "LocalScribe.app", "Contents", "Resources", "bin")
	if dir != want {
		t.Fatalf("dir %s want %s", dir, want)
	}
}

func TestBundledBinDirFromExecutableRejectsOutsideApp(t *testing.T) {
	_, ok := macos.BundledBinDirFromExecutable("/usr/bin/go")
	if ok {
		t.Fatal("expected false outside app")
	}
}

func TestBundledToolPathInsideAppLayout(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "LocalScribe.app", "Contents", "Resources", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(binDir, "whisper-cli")
	if err := os.WriteFile(tool, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	path, ok := macos.BundledBinDirFromExecutable(filepath.Join(root, "LocalScribe.app", "Contents", "MacOS", "LocalScribe"))
	if !ok {
		t.Fatal("bundled dir")
	}
	got, ok := macos.BundledToolPathInDir(path, "whisper-cli")
	if !ok || got != tool {
		t.Fatalf("tool path %q ok=%v", got, ok)
	}
}
