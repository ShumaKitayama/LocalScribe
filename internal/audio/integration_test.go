//go:build integration

package audio

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIntegrationProbeAndConvert(t *testing.T) {
	t.Setenv("LOCALSCRIBE_FFMPEG", "")
	t.Setenv("LOCALSCRIBE_FFPROBE", "")

	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not in PATH")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe not in PATH")
	}

	ctx := t.Context()
	work := t.TempDir()
	input := filepath.Join(work, "input.wav")
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-hide_banner", "-nostdin",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=0.2",
		"-ar", "44100", "-ac", "1",
		input,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate input wav: %v\n%s", err, out)
	}

	prober := Prober{FFprobe: ffprobe, Runner: ExecRunner{}}
	info, err := prober.Probe(ctx, input)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.SampleRate <= 0 || info.Channels <= 0 {
		t.Fatalf("unexpected probe info: %+v", info)
	}

	outWAV := filepath.Join(work, "converted.wav")
	converter := Converter{
		FFmpeg:    ffmpeg,
		Runner:    ExecRunner{},
		freeBytes: AvailableBytes,
	}
	if err := converter.Convert(ctx, work, input, outWAV, info); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	data, err := os.ReadFile(outWAV)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 44 {
		t.Fatalf("output too small: %d bytes", len(data))
	}
}
