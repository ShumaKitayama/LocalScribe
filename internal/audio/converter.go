package audio

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"golang.org/x/sys/unix"
)

const (
	workHeadroomBytes  = 64 * 1024 * 1024
	unknownWorkBytes   = 2 * 1024 * 1024 * 1024
	bytesPerSecondWAV  = 32000
)

// FFmpegArgs は設計書 5 節の ffmpeg 引数を返す。先頭の実行ファイル名は含めない。-y は付けない。
func FFmpegArgs(inputPath, outputWAV string) []string {
	return []string{
		"-hide_banner",
		"-nostdin",
		"-i", inputPath,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		outputWAV,
	}
}

// RequiredWorkBytes は変換前に必要な作業領域のバイト数を返す。長さ不明は 2 GiB。
func RequiredWorkBytes(durationMS int64) int64 {
	if durationMS <= 0 {
		return unknownWorkBytes
	}
	return durationMS/1000*bytesPerSecondWAV + workHeadroomBytes
}

// AvailableBytes は path が載るファイルシステムの利用可能バイト数を返す。
func AvailableBytes(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

// Converter は FFmpeg で 16 kHz mono WAV を作業ディレクトリへ書き出す。
type Converter struct {
	FFmpeg    string
	Runner    CommandRunner
	freeBytes func(string) (int64, error)
}

// NewConverter は ffmpeg のパスを解決して Converter を作る。
func NewConverter(runner CommandRunner) (Converter, error) {
	path, err := ResolveFFmpegPath()
	if err != nil {
		return Converter{}, err
	}
	if runner == nil {
		runner = ExecRunner{}
	}
	return Converter{
		FFmpeg:    path,
		Runner:    runner,
		freeBytes: AvailableBytes,
	}, nil
}

// Convert は空き容量を確認し、作業ディレクトリ内へだけ出力する。
func (c Converter) Convert(ctx context.Context, workDir, inputPath, outputWAV string, probe ProbeInfo) error {
	if err := validateWorkOutput(workDir, outputWAV); err != nil {
		return err
	}
	required := RequiredWorkBytes(probe.DurationMS)
	free, err := c.freeBytes(workDir)
	if err != nil {
		return &domain.UserError{
			Message: "空き容量を確認できませんでした",
			Detail:  err.Error(),
		}
	}
	if free < required {
		return &domain.UserError{
			Message: "作業用の空き容量が足りません",
			Detail:  fmt.Sprintf("need %d bytes, have %d", required, free),
		}
	}
	_, stderr, err := c.Runner.Run(ctx, c.FFmpeg, FFmpegArgs(inputPath, outputWAV))
	if err != nil {
		return &domain.UserError{
			Message: "音声の変換に失敗しました",
			Detail:  strings.TrimSpace(string(stderr)),
		}
	}
	return nil
}

func validateWorkOutput(workDir, outputPath string) error {
	if strings.TrimSpace(workDir) == "" {
		return &domain.UserError{Message: "作業ディレクトリを指定してください", Detail: "empty work dir"}
	}
	if strings.TrimSpace(outputPath) == "" {
		return &domain.UserError{Message: "出力先を指定してください", Detail: "empty output"}
	}
	workAbs, err := filepath.Abs(workDir)
	if err != nil {
		return &domain.UserError{Message: "作業ディレクトリが不正です", Detail: err.Error()}
	}
	outAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return &domain.UserError{Message: "出力先が不正です", Detail: err.Error()}
	}
	rel, err := filepath.Rel(workAbs, outAbs)
	if err != nil {
		return &domain.UserError{Message: "出力先は作業ディレクトリ内にしてください", Detail: err.Error()}
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return &domain.UserError{Message: "出力先は作業ディレクトリ内にしてください", Detail: outAbs}
	}
	return nil
}
