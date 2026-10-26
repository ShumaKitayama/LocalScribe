package domain_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

func validRequest() domain.TranscriptionRequest {
	return domain.TranscriptionRequest{
		SourcePath:    "/tmp/lecture.m4a",
		ModelID:       domain.ModelLargeV3,
		Language:      domain.LanguageJA,
		OutputDir:     "/tmp/out",
		ExportFormats: []domain.ExportFormat{domain.FormatTXT, domain.FormatSRT, domain.FormatJSON},
	}
}

func TestTransitionFromTerminalIsRejected(t *testing.T) {
	t.Parallel()
	terminals := []domain.JobState{
		domain.StateCompleted,
		domain.StateFailed,
		domain.StateCancelled,
		domain.StateInterrupted,
	}
	nextStates := []domain.JobState{
		domain.StateQueued,
		domain.StateValidating,
		domain.StateConverting,
		domain.StateTranscribing,
		domain.StateExporting,
	}
	for _, terminal := range terminals {
		for _, next := range nextStates {
			t.Run(string(terminal)+"->"+string(next), func(t *testing.T) {
				t.Parallel()
				job, err := domain.NewTranscriptionJob("job-1", validRequest())
				if err != nil {
					t.Fatalf("NewTranscriptionJob: %v", err)
				}
				job.State = terminal
				err = job.Transition(next)
				if !errors.Is(err, domain.ErrTerminalJob) {
					t.Fatalf("err = %v, want terminal rejection", err)
				}
				if job.State != terminal {
					t.Fatalf("state = %s, want it to stay %s", job.State, terminal)
				}
			})
		}
	}
}

func TestTransitionAllowsActiveProgress(t *testing.T) {
	t.Parallel()
	job, err := domain.NewTranscriptionJob("job-1", validRequest())
	if err != nil {
		t.Fatalf("NewTranscriptionJob: %v", err)
	}
	steps := []domain.JobState{
		domain.StateValidating,
		domain.StateConverting,
		domain.StateTranscribing,
		domain.StateExporting,
		domain.StateCompleted,
	}
	for _, step := range steps {
		if err := job.Transition(step); err != nil {
			t.Fatalf("Transition(%s): %v", step, err)
		}
	}
	if job.State != domain.StateCompleted {
		t.Fatalf("state = %s", job.State)
	}
}

func TestTransitionRejectsUnknownState(t *testing.T) {
	t.Parallel()
	job, err := domain.NewTranscriptionJob("job-1", validRequest())
	if err != nil {
		t.Fatalf("NewTranscriptionJob: %v", err)
	}
	err = job.Transition("paused")
	if err == nil || errors.Is(err, domain.ErrTerminalJob) {
		t.Fatalf("err = %v", err)
	}
	if job.State != domain.StateQueued {
		t.Fatalf("state = %s", job.State)
	}
}

func TestApplyEditKeepsOriginalText(t *testing.T) {
	t.Parallel()
	transcript := sampleTranscript()
	original := transcript.Segments[0].Text
	if err := transcript.ApplyEdits([]domain.SegmentEdit{{Index: 0, EditedText: ""}}); err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if transcript.Segments[0].Text != original {
		t.Fatalf("text = %q, want original %q", transcript.Segments[0].Text, original)
	}
	if transcript.Segments[0].DisplayText() != "" {
		t.Fatalf("display = %q, want empty edit", transcript.Segments[0].DisplayText())
	}
	if err := transcript.ApplyEdits([]domain.SegmentEdit{{Index: 0, EditedText: "修正後"}}); err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if transcript.Segments[0].Text != original {
		t.Fatalf("text changed to %q", transcript.Segments[0].Text)
	}
	if transcript.Segments[0].DisplayText() != "修正後" {
		t.Fatalf("display = %q", transcript.Segments[0].DisplayText())
	}
}

func TestApplyEditsRejectsBadIndexWithoutChangingEarlierRows(t *testing.T) {
	t.Parallel()
	transcript := sampleTranscript()
	err := transcript.ApplyEdits([]domain.SegmentEdit{
		{Index: 0, EditedText: "先に適用しない"},
		{Index: 3, EditedText: "範囲外"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if transcript.Segments[0].EditedText != nil {
		t.Fatalf("edited = %q", *transcript.Segments[0].EditedText)
	}
}

func TestParseLanguage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		ok   bool
		want domain.Language
	}{
		{in: "ja", ok: true, want: domain.LanguageJA},
		{in: "en", ok: true, want: domain.LanguageEN},
		{in: "auto", ok: true, want: domain.LanguageAuto},
		{in: "JA"},
		{in: "fr"},
		{in: ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseLanguage(tt.in)
			if tt.ok {
				if err != nil {
					t.Fatalf("ParseLanguage: %v", err)
				}
				if got != tt.want {
					t.Fatalf("language = %s", got)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			var userErr *domain.UserError
			if !errors.As(err, &userErr) {
				t.Fatalf("err = %T", err)
			}
			if userErr.Message == "" || userErr.Detail != tt.in {
				t.Fatalf("message = %q detail = %q", userErr.Message, userErr.Detail)
			}
		})
	}
}

func TestExtensions(t *testing.T) {
	t.Parallel()
	accepted := []string{"a.mp3", "b.M4A", "c.wav", "d.FLAC", "e.mp4", "f.mov"}
	for _, path := range accepted {
		if !domain.AllowedUserExtension(path) {
			t.Fatalf("%s should be accepted", path)
		}
		if domain.AllowedRecordingExtension(path) {
			t.Fatalf("%s should not be a recording", path)
		}
	}
	rejected := []string{"a.txt", "a.webm", "a", "dir/"}
	for _, path := range rejected {
		if domain.AllowedUserExtension(path) {
			t.Fatalf("%s should be rejected", path)
		}
	}
	if !domain.AllowedRecordingExtension("take.WEBM") {
		t.Fatal("webm recording should be accepted")
	}
}

func TestRequestRejectsRecordingWithUserExtension(t *testing.T) {
	t.Parallel()
	req := validRequest()
	req.SourceKind = domain.SourceRecording
	if err := req.Validate(); err == nil {
		t.Fatal("m4a recording should be rejected")
	}
	req.SourcePath = "/tmp/take.webm"
	if err := req.Validate(); err != nil {
		t.Fatalf("webm recording: %v", err)
	}
	req.SourceKind = domain.SourceFile
	if err := req.Validate(); err == nil {
		t.Fatal("webm file drop should be rejected")
	}
}

func TestPromptLimit(t *testing.T) {
	t.Parallel()
	req := validRequest()
	req.InitialPrompt = strings.Repeat("あ", domain.MaxPromptRunes)
	if err := req.Validate(); err != nil {
		t.Fatalf("500 runes: %v", err)
	}
	req.InitialPrompt = strings.Repeat("あ", domain.MaxPromptRunes+1)
	if err := req.Validate(); err == nil {
		t.Fatal("501 runes should be rejected")
	}
}

func TestProgressJSONOmitsUnknownPercentAndKeepsZero(t *testing.T) {
	t.Parallel()
	unknown, err := json.Marshal(domain.JobProgress{
		JobID:     "job-1",
		State:     domain.StateTranscribing,
		ElapsedMS: 12000,
		Message:   "音声を認識しています",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(unknown), "percent") {
		t.Fatalf("unknown percent was serialized: %s", unknown)
	}
	zero := 0.0
	measured, err := json.Marshal(domain.JobProgress{Percent: &zero})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(measured), `"percent":0`) {
		t.Fatalf("zero percent = %s", measured)
	}
}

func TestSegmentJSONKeepsNullEditedText(t *testing.T) {
	t.Parallel()
	body, err := json.Marshal(domain.Segment{StartMS: 0, EndMS: 1200, Text: "認識直後の文"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(body), `"edited_text":null`) {
		t.Fatalf("body = %s", body)
	}
}

func TestNewIDUsesGeneratorAndRandomHex(t *testing.T) {
	t.Parallel()
	got, err := domain.NewID(func() (string, error) { return "fixed-id", nil })
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	if got != "fixed-id" {
		t.Fatalf("id = %s", got)
	}
	randomID, err := domain.NewID(nil)
	if err != nil {
		t.Fatalf("RandomID: %v", err)
	}
	if len(randomID) != 32 {
		t.Fatalf("len = %d, id = %s", len(randomID), randomID)
	}
}

func TestModelMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id       string
		file     string
		minBytes int64
	}{
		{id: domain.ModelLargeV3, file: "ggml-large-v3.bin", minBytes: 2_800_000_000},
		{id: domain.ModelLargeV3Turbo, file: "ggml-large-v3-turbo.bin", minBytes: 1_400_000_000},
	}
	for _, tt := range tests {
		name, ok := domain.ModelFileName(tt.id)
		if !ok || name != tt.file {
			t.Fatalf("file = %s ok = %v", name, ok)
		}
		size, ok := domain.ModelMinBytes(tt.id)
		if !ok || size != tt.minBytes {
			t.Fatalf("min = %d ok = %v", size, ok)
		}
	}
	if domain.ValidModelID("tiny") {
		t.Fatal("tiny should be rejected")
	}
}

func TestJobOmitsZeroTimes(t *testing.T) {
	t.Parallel()
	job, err := domain.NewTranscriptionJob("job-1", validRequest())
	if err != nil {
		t.Fatalf("NewTranscriptionJob: %v", err)
	}
	body, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(body), "started_at") || strings.Contains(string(body), "ended_at") {
		t.Fatalf("zero times were serialized: %s", body)
	}
}

func sampleTranscript() domain.Transcript {
	return domain.Transcript{
		SchemaVersion: domain.SchemaVersion,
		ID:            "tr-1",
		JobID:         "job-1",
		ModelID:       domain.ModelLargeV3,
		Language:      domain.LanguageJA,
		SourcePath:    "/tmp/lecture.m4a",
		DurationMS:    1200,
		CreatedAt:     time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC),
		Segments: []domain.Segment{
			{StartMS: 0, EndMS: 1200, Text: "認識直後の文"},
		},
	}
}
