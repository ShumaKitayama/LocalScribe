package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

const transcriptFileName = "transcript.json"

// PrepareOutputDir は {stem}_{YYYYMMDD}_{連番} の空きディレクトリを作って返す。既存は上書きしない。
func PrepareOutputDir(outputDir, sourcePath string, now time.Time) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", &domain.UserError{Message: "保存先を指定してください", Detail: "empty output dir"}
	}
	if strings.TrimSpace(sourcePath) == "" {
		return "", &domain.UserError{Message: "入力ファイルを指定してください", Detail: "empty source path"}
	}
	stem := transcriptStem(sourcePath)
	date := now.Format("20060102")
	for seq := 1; seq < 10000; seq++ {
		name := fmt.Sprintf("%s_%s_%03d", stem, date, seq)
		full := filepath.Join(outputDir, name)
		_, statErr := os.Stat(full)
		if statErr == nil {
			continue
		}
		if !os.IsNotExist(statErr) {
			return "", fmt.Errorf("stat output dir %q: %w", full, statErr)
		}
		if err := os.Mkdir(full, 0o755); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", fmt.Errorf("create output dir %q: %w", full, err)
		}
		return full, nil
	}
	return "", &domain.UserError{Message: "保存先の連番を使い切りました", Detail: outputDir}
}

// SaveTranscript は transcript.json を原子的に書く。dir は既に存在していること。
func SaveTranscript(dir string, transcript domain.Transcript) error {
	if strings.TrimSpace(dir) == "" {
		return &domain.UserError{Message: "保存先がありません", Detail: "empty dir"}
	}
	if err := transcript.Validate(); err != nil {
		return err
	}
	path := filepath.Join(dir, transcriptFileName)
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return fmt.Errorf("encode transcript: %w", err)
	}
	data = append(data, '\n')
	if err := WriteFileAtomic(path, data); err != nil {
		return err
	}
	return nil
}

// LoadTranscript は transcript.json を読む。
func LoadTranscript(dir string) (domain.Transcript, error) {
	path := filepath.Join(dir, transcriptFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Transcript{}, fmt.Errorf("read transcript: %w", err)
	}
	var transcript domain.Transcript
	if err := json.Unmarshal(data, &transcript); err != nil {
		return domain.Transcript{}, fmt.Errorf("decode transcript: %w", err)
	}
	return transcript, nil
}

func transcriptStem(sourcePath string) string {
	base := filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" || stem == "." {
		return "transcript"
	}
	return stem
}
