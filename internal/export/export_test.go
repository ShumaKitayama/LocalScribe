package export

import (
	"strings"
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

func testTranscript(segments []domain.Segment) domain.Transcript {
	return domain.Transcript{
		SchemaVersion: domain.SchemaVersion,
		ID:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		JobID:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ModelID:       domain.ModelLargeV3,
		Language:      domain.LanguageJA,
		SourcePath:    "/tmp/lecture.m4a",
		DurationMS:    3_601_000,
		CreatedAt:     time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC),
		Segments:      segments,
	}
}

func TestRenderTXTAndSRTFromJSONGolden(t *testing.T) {
	edited := "編集後の文"
	segments := []domain.Segment{
		{StartMS: 0, EndMS: 0, Text: "ゼロミリ秒"},
		{StartMS: 3_599_000, EndMS: 3_601_000, Text: "認識直後", EditedText: &edited},
	}
	transcript := testTranscript(segments)

	jsonData, err := EncodeTranscriptJSON(transcript)
	if err != nil {
		t.Fatalf("EncodeTranscriptJSON: %v", err)
	}

	txt, err := TXTFromJSON(jsonData)
	if err != nil {
		t.Fatalf("TXTFromJSON: %v", err)
	}
	wantTXT := "ゼロミリ秒\n編集後の文"
	if txt != wantTXT {
		t.Fatalf("TXT:\n%s\nwant:\n%s", txt, wantTXT)
	}

	srt, err := SRTFromJSON(jsonData)
	if err != nil {
		t.Fatalf("SRTFromJSON: %v", err)
	}
	wantSRT := strings.Join([]string{
		"1",
		"00:00:00,000 --> 00:00:00,000",
		"ゼロミリ秒",
		"",
		"2",
		"00:59:59,000 --> 01:00:01,000",
		"編集後の文",
		"",
	}, "\n") + "\n"
	if srt != wantSRT {
		t.Fatalf("SRT:\n%s\nwant:\n%s", srt, wantSRT)
	}
}

func TestDecodeTranscriptJSONRequiresSchemaVersion(t *testing.T) {
	transcript := testTranscript(nil)
	transcript.SchemaVersion = 99
	data, err := EncodeTranscriptJSON(transcript)
	if err == nil {
		t.Fatal("expected encode validation error")
	}
	_ = data
	raw := []byte(`{"schema_version":99,"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","job_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","model_id":"large-v3","language":"ja","source_path":"/x","duration_ms":0,"created_at":"2026-10-27T00:00:00Z","segments":[]}`)
	_, err = DecodeTranscriptJSON(raw)
	if err == nil {
		t.Fatal("expected decode validation error")
	}
}

func TestRenderSRTRoundTripThroughJSON(t *testing.T) {
	transcript := testTranscript([]domain.Segment{
		{StartMS: 0, EndMS: 500, Text: "hello"},
	})
	jsonData, err := EncodeTranscriptJSON(transcript)
	if err != nil {
		t.Fatal(err)
	}
	direct := RenderSRT(transcript)
	viaJSON, err := SRTFromJSON(jsonData)
	if err != nil {
		t.Fatal(err)
	}
	if direct != viaJSON {
		t.Fatalf("SRT mismatch:\ndirect %q\njson %q", direct, viaJSON)
	}
}
