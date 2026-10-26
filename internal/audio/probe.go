package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

// ProbeInfo は FFprobe から取った音声メタデータである。
type ProbeInfo struct {
	DurationMS int64
	SampleRate int
	Channels   int
}

// FFprobeArgs は設計書 5 節の ffprobe 引数を返す。先頭の実行ファイル名は含めない。
func FFprobeArgs(inputPath string) []string {
	return []string{
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}
}

// Prober は FFprobe を呼び出して入力を調べる。
type Prober struct {
	FFprobe string
	Runner  CommandRunner
}

// NewProber は ffprobe のパスを解決して Prober を作る。
func NewProber(runner CommandRunner) (Prober, error) {
	path, err := ResolveFFprobePath()
	if err != nil {
		return Prober{}, err
	}
	if runner == nil {
		runner = ExecRunner{}
	}
	return Prober{FFprobe: path, Runner: runner}, nil
}

// Probe は入力の長さと音声ストリームを調べる。音声が無いときはエラーにする。
func (p Prober) Probe(ctx context.Context, inputPath string) (ProbeInfo, error) {
	if strings.TrimSpace(inputPath) == "" {
		return ProbeInfo{}, &domain.UserError{Message: "入力ファイルを指定してください", Detail: "empty input"}
	}
	stdout, stderr, err := p.Runner.Run(ctx, p.FFprobe, FFprobeArgs(inputPath))
	if err != nil {
		return ProbeInfo{}, &domain.UserError{
			Message: "音声ファイルを調べられませんでした",
			Detail:  strings.TrimSpace(string(stderr)),
		}
	}
	return ParseFFprobeJSON(stdout)
}

type ffprobePayload struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType  string `json:"codec_type"`
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

// ParseFFprobeJSON は ffprobe の JSON を解釈する。
func ParseFFprobeJSON(data []byte) (ProbeInfo, error) {
	var payload ffprobePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return ProbeInfo{}, &domain.UserError{
			Message: "音声ファイルの情報を読めませんでした",
			Detail:  err.Error(),
		}
	}
	var audio *ffprobeStream
	for i := range payload.Streams {
		if payload.Streams[i].CodecType == "audio" {
			audio = &payload.Streams[i]
			break
		}
	}
	if audio == nil {
		return ProbeInfo{}, &domain.UserError{
			Message: "音声トラックがありません",
			Detail:  "no audio stream",
		}
	}
	rate, err := strconv.Atoi(audio.SampleRate)
	if err != nil || rate <= 0 {
		return ProbeInfo{}, &domain.UserError{
			Message: "音声ファイルの情報を読めませんでした",
			Detail:  "invalid sample_rate",
		}
	}
	if audio.Channels <= 0 {
		return ProbeInfo{}, &domain.UserError{
			Message: "音声ファイルの情報を読めませんでした",
			Detail:  "invalid channels",
		}
	}
	durationMS, err := parseDurationMS(payload.Format.Duration)
	if err != nil {
		return ProbeInfo{}, err
	}
	return ProbeInfo{
		DurationMS: durationMS,
		SampleRate: rate,
		Channels:   audio.Channels,
	}, nil
}

func parseDurationMS(value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0, &domain.UserError{
			Message: "音声ファイルの情報を読めませんでした",
			Detail:  fmt.Sprintf("invalid duration: %q", value),
		}
	}
	return int64(math.Round(seconds * 1000)), nil
}
