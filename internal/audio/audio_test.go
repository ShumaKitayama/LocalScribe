package audio

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

type stubRunner struct {
	lastBinary string
	lastArgs   []string
	stdout     []byte
	stderr     []byte
	err        error
}

func (s *stubRunner) Run(ctx context.Context, binary string, args []string) ([]byte, []byte, error) {
	s.lastBinary = binary
	s.lastArgs = args
	return s.stdout, s.stderr, s.err
}

func TestFFmpegArgsOrder(t *testing.T) {
	got := FFmpegArgs("/in/lecture.m4a", "/work/job.wav")
	want := []string{
		"-hide_banner",
		"-nostdin",
		"-i", "/in/lecture.m4a",
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"/work/job.wav",
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

func TestFFprobeArgsOrder(t *testing.T) {
	got := FFprobeArgs("/in/lecture.m4a")
	want := []string{
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"/in/lecture.m4a",
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

func TestParseFFprobeJSONNoAudioStream(t *testing.T) {
	_, err := ParseFFprobeJSON([]byte(`{"streams":[{"codec_type":"video"}]}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*domain.UserError); !ok {
		t.Fatalf("expected UserError, got %T", err)
	}
}

func TestRequiredWorkBytes(t *testing.T) {
	if RequiredWorkBytes(0) != unknownWorkBytes {
		t.Fatalf("unknown duration")
	}
	const durationMS = 10_000
	want := int64(durationMS/1000*bytesPerSecondWAV + workHeadroomBytes)
	if RequiredWorkBytes(durationMS) != want {
		t.Fatalf("got %d want %d", RequiredWorkBytes(durationMS), want)
	}
}

func TestConvertRejectsOutputOutsideWorkDir(t *testing.T) {
	runner := &stubRunner{}
	c := Converter{FFmpeg: "ffmpeg", Runner: runner, freeBytes: func(string) (int64, error) {
		return unknownWorkBytes, nil
	}}
	work := t.TempDir()
	outside := filepath.Join(t.TempDir(), "out.wav")
	err := c.Convert(context.Background(), work, "/in/file.m4a", outside, ProbeInfo{DurationMS: 1000, SampleRate: 44100, Channels: 2})
	if err == nil {
		t.Fatal("expected error")
	}
	if runner.lastArgs != nil {
		t.Fatal("runner should not be called")
	}
}

func TestConvertRejectsInsufficientSpace(t *testing.T) {
	runner := &stubRunner{}
	c := Converter{FFmpeg: "ffmpeg", Runner: runner, freeBytes: func(string) (int64, error) {
		return 1, nil
	}}
	work := t.TempDir()
	out := filepath.Join(work, "job.wav")
	err := c.Convert(context.Background(), work, "/in/file.m4a", out, ProbeInfo{DurationMS: 60_000, SampleRate: 44100, Channels: 2})
	if err == nil {
		t.Fatal("expected error")
	}
	if runner.lastArgs != nil {
		t.Fatal("runner should not be called when space is insufficient")
	}
}

func TestProbeUsesStubWithoutTouchingInputFile(t *testing.T) {
	runner := &stubRunner{
		stdout: []byte(`{
			"streams":[{"codec_type":"audio","sample_rate":"48000","channels":2}],
			"format":{"duration":"1.5"}
		}`),
	}
	p := Prober{FFprobe: "ffprobe", Runner: runner}
	input := filepath.Join(t.TempDir(), "does-not-exist.m4a")
	info, err := p.Probe(context.Background(), input)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.DurationMS != 1500 || info.SampleRate != 48000 || info.Channels != 2 {
		t.Fatalf("unexpected info: %+v", info)
	}
	if _, err := os.Stat(input); !os.IsNotExist(err) {
		t.Fatal("input file should not be created by stub probe")
	}
	if runner.lastBinary != "ffprobe" {
		t.Fatalf("binary %q", runner.lastBinary)
	}
}

func TestConvertUsesStubWithoutTouchingInputFile(t *testing.T) {
	runner := &stubRunner{}
	c := Converter{
		FFmpeg: "ffmpeg",
		Runner: runner,
		freeBytes: func(string) (int64, error) {
			return unknownWorkBytes, nil
		},
	}
	work := t.TempDir()
	input := filepath.Join(t.TempDir(), "missing-source.m4a")
	out := filepath.Join(work, "job.wav")
	if err := c.Convert(context.Background(), work, input, out, ProbeInfo{DurationMS: 500, SampleRate: 44100, Channels: 1}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if _, err := os.Stat(input); !os.IsNotExist(err) {
		t.Fatal("stub convert must not read or create the source file")
	}
	if runner.lastArgs == nil {
		t.Fatal("runner not called")
	}
}
