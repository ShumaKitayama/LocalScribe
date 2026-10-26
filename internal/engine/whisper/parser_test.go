package whisper

import (
	"os"
	"path/filepath"
	"testing"
)

func fixtureWhisperOutput(t *testing.T) []byte {
	path := filepath.Join("..", "..", "..", "tests", "fixtures", "whisper-output.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

func TestParseOutputJSONFixture(t *testing.T) {
	segments, err := ParseOutputJSON(fixtureWhisperOutput(t))
	if err != nil {
		t.Fatalf("ParseOutputJSON: %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("len %d want 1", len(segments))
	}
	seg := segments[0]
	if seg.StartMS != 990 || seg.EndMS != 7800 {
		t.Fatalf("offsets: start=%d end=%d", seg.StartMS, seg.EndMS)
	}
	wantText := "My fellow Americans, this day has brought terrible news and great sadness to our country."
	if seg.Text != wantText {
		t.Fatalf("text %q", seg.Text)
	}
}

func TestParseOutputJSONEmptyTranscription(t *testing.T) {
	raw := []byte(`{"transcription":[]}`)
	segments, err := ParseOutputJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 0 {
		t.Fatalf("len %d", len(segments))
	}
}
