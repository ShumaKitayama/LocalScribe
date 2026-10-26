package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
)

// SettingsStore は settings.json の読み書きを行う。
type SettingsStore struct {
	path string
	home string
}

// NewSettingsStore は home から settings.json のパスを決めてストアを作る。
func NewSettingsStore(home string) (SettingsStore, error) {
	if err := validateHome(home); err != nil {
		return SettingsStore{}, err
	}
	return SettingsStore{
		path: macos.SettingsPath(home),
		home: home,
	}, nil
}

// DefaultSettings は設計書の初期値を返す。保存はしない。
func DefaultSettings(home string) (domain.AppSettings, error) {
	if err := validateHome(home); err != nil {
		return domain.AppSettings{}, err
	}
	return domain.AppSettings{
		SchemaVersion:  domain.SchemaVersion,
		DefaultModelID: domain.ModelLargeV3,
		Language:       domain.LanguageJA,
		OutputDir:      macos.DefaultTranscriptsDir(home),
		ExportFormats: []domain.ExportFormat{
			domain.FormatTXT,
			domain.FormatSRT,
			domain.FormatJSON,
		},
		InitialPrompt: "",
	}, nil
}

// Load は設定を読む。ファイルが無いときは既定値を返す。
func (s SettingsStore) Load() (domain.AppSettings, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultSettings(s.home)
		}
		return domain.AppSettings{}, fmt.Errorf("read settings: %w", err)
	}
	var settings domain.AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return domain.AppSettings{}, fmt.Errorf("decode settings: %w", err)
	}
	return settings, nil
}

// Save は output_dir の ~ を展開してから原子的に保存する。
func (s SettingsStore) Save(settings domain.AppSettings) error {
	expanded, err := macos.ExpandHome(s.home, settings.OutputDir)
	if err != nil {
		return err
	}
	settings.OutputDir = expanded
	if err := settings.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	data = append(data, '\n')
	if err := WriteFileAtomic(s.path, data); err != nil {
		return err
	}
	return nil
}

func validateHome(home string) error {
	if strings.TrimSpace(home) == "" {
		return &domain.UserError{Message: "ホームディレクトリがありません", Detail: "empty home"}
	}
	return nil
}
