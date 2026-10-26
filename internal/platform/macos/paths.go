// Package macos は macOS 上の LocalScribe 用パスを解決する。
package macos

import (
	"path/filepath"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

const (
	appSupportName = "LocalScribe"
)

// ApplicationSupportDir は ~/Library/Application Support/LocalScribe を返す。
func ApplicationSupportDir(home string) string {
	return filepath.Join(home, "Library", "Application Support", appSupportName)
}

// ModelsDir はモデルファイルの置き場を返す。
func ModelsDir(home string) string {
	return filepath.Join(ApplicationSupportDir(home), "models")
}

// BinDir は同梱バイナリの置き場を返す。
func BinDir(home string) string {
	return filepath.Join(ApplicationSupportDir(home), "bin")
}

// JobsDir はジョブ JSON の置き場を返す。
func JobsDir(home string) string {
	return filepath.Join(ApplicationSupportDir(home), "jobs")
}

// SettingsPath は settings.json のパスを返す。
func SettingsPath(home string) string {
	return filepath.Join(ApplicationSupportDir(home), "settings.json")
}

// WorkCacheDir は作業 WAV 用のキャッシュを返す。
func WorkCacheDir(home string) string {
	return filepath.Join(home, "Library", "Caches", appSupportName, "work")
}

// LogPath はアプリログのパスを返す。
func LogPath(home string) string {
	return filepath.Join(home, "Library", "Logs", appSupportName, "localscribe.log")
}

// DefaultTranscriptsDir は文字起こし結果の既定保存先を返す。
func DefaultTranscriptsDir(home string) string {
	return filepath.Join(home, "Documents", appSupportName, "Transcripts")
}

// ExpandHome は先頭の ~ を home で置き換える。それ以外のパスはそのまま返す。
func ExpandHome(home, path string) (string, error) {
	if strings.TrimSpace(home) == "" {
		return "", &domain.UserError{Message: "ホームディレクトリがありません", Detail: "empty home"}
	}
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", &domain.UserError{Message: "パスを指定してください", Detail: "empty path"}
	}
	if trimmed == "~" {
		return home, nil
	}
	if strings.HasPrefix(trimmed, "~/") {
		return filepath.Join(home, trimmed[2:]), nil
	}
	if strings.HasPrefix(trimmed, "~") {
		return "", &domain.UserError{Message: "この形式の ~ は展開できません", Detail: trimmed}
	}
	return trimmed, nil
}
