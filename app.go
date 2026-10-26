package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
	"github.com/ShumaKitayama/localscribe/internal/storage"
)

// App はフェーズ 9 のマイク録音確認用 Wails API である。
type App struct {
	ctx  context.Context
	home string
}

// SaveRecordingResult は Go 側へ保存した録音ファイルの情報である。
type SaveRecordingResult struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

// NewApp は Wails 用のアプリ構造体を作る。
func NewApp() *App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return &App{home: home}
}

// startup は Wails 起動時に呼ばれる。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SaveRecording は WebM の録音バイト列をキャッシュへ保存し、絶対パスを返す。
func (a *App) SaveRecording(data []byte) (SaveRecordingResult, error) {
	if len(data) == 0 {
		return SaveRecordingResult{}, &domain.UserError{
			Message: "録音データがありません",
			Detail:  "empty recording",
		}
	}
	if a.home == "" {
		return SaveRecordingResult{}, &domain.UserError{
			Message: "ホームディレクトリがありません",
			Detail:  "empty home",
		}
	}
	dir := filepath.Join(macos.WorkCacheDir(a.home), "recordings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SaveRecordingResult{}, &domain.UserError{
			Message: "録音の保存先を作れません",
			Detail:  err.Error(),
		}
	}
	name := fmt.Sprintf("recording_%d.webm", time.Now().UnixMilli())
	path := filepath.Join(dir, name)
	if err := storage.WriteFileAtomic(path, data); err != nil {
		return SaveRecordingResult{}, &domain.UserError{
			Message: "録音を保存できません",
			Detail:  err.Error(),
		}
	}
	return SaveRecordingResult{
		Path:      path,
		SizeBytes: int64(len(data)),
	}, nil
}
