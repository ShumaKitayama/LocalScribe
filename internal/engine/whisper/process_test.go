package whisper

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/engine"
)

type blockingRunner struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingRunner) Run(ctx context.Context, binary string, args []string, onStderrLine func(string)) error {
	close(b.started)
	select {
	case <-ctx.Done():
		return engine.ErrCancelled
	case <-b.release:
		return nil
	}
}

func TestTranscribeCancelReturnsCancelled(t *testing.T) {
	runner := &blockingRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	eng := Engine{Binary: "whisper-cli", Runner: runner}
	ctx, cancel := context.WithCancel(context.Background())
	req := engine.Request{
		WAVPath:      "/tmp/in.wav",
		ModelPath:    "/tmp/model.bin",
		Language:     domain.LanguageJA,
		OutputPrefix: filepath.Join(t.TempDir(), "whisper-out"),
	}
	go func() {
		<-runner.started
		cancel()
	}()
	_, err := eng.Transcribe(ctx, req, nil)
	if !engine.IsCancelled(err) {
		t.Fatalf("expected cancelled, got %v", err)
	}
}

func TestParseProgressLineWithoutPercentKeepsNil(t *testing.T) {
	update := ParseProgressLine("whisper_init_from_file_with_state: loading model")
	if update.Percent != nil {
		t.Fatalf("expected nil percent, got %v", *update.Percent)
	}
	if update.Message == "" {
		t.Fatal("expected message")
	}
}

func TestParseProgressLineWithPercent(t *testing.T) {
	update := ParseProgressLine("whisper_print_progress_callback: progress =  42%")
	if update.Percent == nil {
		t.Fatal("expected percent")
	}
	if *update.Percent != 42 {
		t.Fatalf("percent %v", *update.Percent)
	}
}

func TestTranscribeProgressWithoutPercentStaysNil(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "whisper-out")
	jsonPath := prefix + ".json"
	if err := os.WriteFile(jsonPath, fixtureWhisperOutput(t), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &immediateRunner{
		stderrLines: []string{"loading", "whisper_print_progress_callback: note without percent"},
	}
	eng := Engine{Binary: "whisper-cli", Runner: runner}
	var last engine.Progress
	_, err := eng.Transcribe(context.Background(), engine.Request{
		WAVPath:      "/tmp/in.wav",
		ModelPath:    "/tmp/model.bin",
		Language:     domain.LanguageJA,
		OutputPrefix: prefix,
	}, func(p engine.Progress) {
		last = p
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if last.Percent != nil {
		t.Fatalf("expected nil percent on last update, got %v", *last.Percent)
	}
}

type immediateRunner struct {
	stderrLines []string
}

func (r *immediateRunner) Run(ctx context.Context, binary string, args []string, onStderrLine func(string)) error {
	for _, line := range r.stderrLines {
		if onStderrLine != nil {
			onStderrLine(line)
		}
	}
	return nil
}

func TestArgsOrderMatchesDesign(t *testing.T) {
	req := engine.Request{
		WAVPath:       "/work/in.wav",
		ModelPath:     "/models/ggml-large-v3.bin",
		Language:      domain.LanguageJA,
		InitialPrompt: "専門用語",
		OutputPrefix:  "/work/whisper-out",
	}
	got := Args(req)
	want := []string{
		"-m", "/models/ggml-large-v3.bin",
		"-f", "/work/in.wav",
		"-l", "ja",
		"-oj",
		"-of", "/work/whisper-out",
		"-pp",
		"--prompt", "専門用語",
	}
	if len(got) != len(want) {
		t.Fatalf("len %d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestArgsOmitsPromptWhenEmpty(t *testing.T) {
	req := engine.Request{
		WAVPath:      "/work/in.wav",
		ModelPath:    "/models/ggml-large-v3.bin",
		Language:     domain.LanguageAuto,
		OutputPrefix: "/work/whisper-out",
	}
	got := Args(req)
	if len(got) != 10 {
		t.Fatalf("unexpected args: %v", got)
	}
	if got[5] != "auto" {
		t.Fatalf("language flag %q", got[5])
	}
}

func TestBlockingRunnerRespectsTimeoutCancel(t *testing.T) {
	runner := &blockingRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := runner.Run(ctx, "whisper-cli", nil, nil)
	if !engine.IsCancelled(err) {
		t.Fatalf("expected cancelled, got %v", err)
	}
}
