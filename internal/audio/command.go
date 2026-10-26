package audio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

// CommandRunner は外部コマンドの標準出力と標準エラーを返す。テストでは偽実装を渡す。
type CommandRunner interface {
	Run(ctx context.Context, binary string, args []string) (stdout []byte, stderr []byte, err error)
}

// ExecRunner は exec.CommandContext でコマンドを起動する。
type ExecRunner struct{}

// Run は binary と引数配列でプロセスを起動する。
func (ExecRunner) Run(ctx context.Context, binary string, args []string) (stdout []byte, stderr []byte, err error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return outBuf.Bytes(), errBuf.Bytes(), fmt.Errorf("%s: %w", binary, err)
	}
	return outBuf.Bytes(), errBuf.Bytes(), nil
}

// ResolveFFmpegPath は LOCALSCRIBE_FFMPEG または PATH から ffmpeg を探す。
func ResolveFFmpegPath() (string, error) {
	if path := os.Getenv("LOCALSCRIBE_FFMPEG"); path != "" {
		return path, nil
	}
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", &domain.UserError{
			Message: "FFmpeg が見つかりません",
			Detail:  err.Error(),
		}
	}
	return path, nil
}

// ResolveFFprobePath は LOCALSCRIBE_FFPROBE または PATH から ffprobe を探す。
func ResolveFFprobePath() (string, error) {
	if path := os.Getenv("LOCALSCRIBE_FFPROBE"); path != "" {
		return path, nil
	}
	path, err := exec.LookPath("ffprobe")
	if err != nil {
		return "", &domain.UserError{
			Message: "FFprobe が見つかりません",
			Detail:  err.Error(),
		}
	}
	return path, nil
}
