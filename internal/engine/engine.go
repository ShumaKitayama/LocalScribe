package engine

import (
	"context"
	"errors"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

// ErrCancelled は context 取消で推論が止まったときの識別子である。
var ErrCancelled = errors.New("engine: cancelled")

// Progress は標準エラーから読んだ進捗。Percent が nil のときは未計測であり 0 を入れない。
type Progress struct {
	Message string
	Percent *float64
}

// Request は whisper 推論に渡す入力である。
type Request struct {
	WAVPath       string
	ModelPath     string
	Language      domain.Language
	InitialPrompt string
	WorkDir       string
	OutputPrefix  string
}

// Engine は音声認識エンジンの差し替え点である。MVP の実装は whisper のみ。
type Engine interface {
	Transcribe(ctx context.Context, req Request, onProgress func(Progress)) ([]domain.Segment, error)
}

// IsCancelled は取消で終わったエラーかを返す。
func IsCancelled(err error) bool {
	return errors.Is(err, ErrCancelled)
}
