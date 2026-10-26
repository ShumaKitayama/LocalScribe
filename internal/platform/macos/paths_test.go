package macos

import (
	"path/filepath"
	"testing"
)

func TestPathsUnderHome(t *testing.T) {
	home := "/Users/tester"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"application support", ApplicationSupportDir(home), filepath.Join(home, "Library", "Application Support", "LocalScribe")},
		{"models", ModelsDir(home), filepath.Join(home, "Library", "Application Support", "LocalScribe", "models")},
		{"jobs", JobsDir(home), filepath.Join(home, "Library", "Application Support", "LocalScribe", "jobs")},
		{"settings", SettingsPath(home), filepath.Join(home, "Library", "Application Support", "LocalScribe", "settings.json")},
		{"work cache", WorkCacheDir(home), filepath.Join(home, "Library", "Caches", "LocalScribe", "work")},
		{"log", LogPath(home), filepath.Join(home, "Library", "Logs", "LocalScribe", "localscribe.log")},
		{"default transcripts", DefaultTranscriptsDir(home), filepath.Join(home, "Documents", "LocalScribe", "Transcripts")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %q want %q", tc.got, tc.want)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home := "/Users/tester"
	got, err := ExpandHome(home, "~/Documents/LocalScribe/Transcripts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "Documents", "LocalScribe", "Transcripts")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	abs, err := ExpandHome(home, "/tmp/out")
	if err != nil || abs != "/tmp/out" {
		t.Fatalf("absolute path: got %q err %v", abs, err)
	}
}
