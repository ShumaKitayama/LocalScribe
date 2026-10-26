package domain

import (
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

// MaxPromptRunes は初期プロンプトの上限である。
const MaxPromptRunes = 500

// Language は認識言語である。MVP では ja、en、auto だけを許す。
type Language string

const (
	LanguageJA   Language = "ja"
	LanguageEN   Language = "en"
	LanguageAuto Language = "auto"
)

// Valid は許容された言語なら true を返す。
func (l Language) Valid() bool {
	return slices.Contains([]Language{LanguageJA, LanguageEN, LanguageAuto}, l)
}

// ParseLanguage は保存用の言語へ変換する。大文字や未対応の値は拒否する。
func ParseLanguage(value string) (Language, error) {
	lang := Language(value)
	if !lang.Valid() {
		return "", &UserError{Message: "対応していない言語です", Detail: value}
	}
	return lang, nil
}

// ExportFormat はユーザーへ書き出す形式である。
type ExportFormat string

const (
	FormatTXT  ExportFormat = "txt"
	FormatSRT  ExportFormat = "srt"
	FormatJSON ExportFormat = "json"
)

// Valid は txt、srt、json のいずれかなら true を返す。
func (f ExportFormat) Valid() bool {
	return slices.Contains([]ExportFormat{FormatTXT, FormatSRT, FormatJSON}, f)
}

// ValidateFormats は空と重複と未知の形式を拒否する。
func ValidateFormats(formats []ExportFormat) error {
	if len(formats) == 0 {
		return &UserError{Message: "保存形式を1つ以上選んでください", Detail: "empty export formats"}
	}
	seen := make(map[ExportFormat]struct{}, len(formats))
	for _, format := range formats {
		if !format.Valid() {
			return &UserError{Message: "対応していない保存形式です", Detail: string(format)}
		}
		if _, exists := seen[format]; exists {
			return &UserError{Message: "保存形式が重複しています", Detail: string(format)}
		}
		seen[format] = struct{}{}
	}
	return nil
}

// ValidatePrompt は 500 文字を超える初期プロンプトを拒否する。
func ValidatePrompt(prompt string) error {
	if utf8.RuneCountInString(prompt) > MaxPromptRunes {
		return &UserError{Message: "初期プロンプトは500文字以内にしてください", Detail: "prompt too long"}
	}
	return nil
}

// SourceKind はファイル選択かマイク録音かを分ける。空はファイル選択として扱う。
type SourceKind string

const (
	SourceFile      SourceKind = "file"
	SourceRecording SourceKind = "recording"
)

// Valid は空、file、recording なら true を返す。
func (k SourceKind) Valid() bool {
	return k == "" || k == SourceFile || k == SourceRecording
}

var userInputExts = []string{".mp3", ".m4a", ".wav", ".flac", ".mp4", ".mov"}

// AllowedUserExtension はドロップと Finder 選択で受け付ける拡張子かを返す。
func AllowedUserExtension(path string) bool {
	return slices.Contains(userInputExts, strings.ToLower(filepath.Ext(path)))
}

// AllowedRecordingExtension はマイク録音の保存形式かを返す。webm だけを許す。
func AllowedRecordingExtension(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".webm")
}

// AppSettings はアプリの既定値である。
type AppSettings struct {
	SchemaVersion  int            `json:"schema_version"`
	DefaultModelID string         `json:"default_model_id"`
	Language       Language       `json:"language"`
	OutputDir      string         `json:"output_dir"`
	ExportFormats  []ExportFormat `json:"export_formats"`
	InitialPrompt  string         `json:"initial_prompt"`
}

// Validate は設定を保存してよいかを判定する。ディレクトリの実在は見ない。
func (s AppSettings) Validate() error {
	if s.SchemaVersion != SchemaVersion {
		return &UserError{Message: "設定の形式が対応外です", Detail: "schema_version"}
	}
	if !ValidModelID(s.DefaultModelID) {
		return &UserError{Message: "対応していないモデルです", Detail: s.DefaultModelID}
	}
	if !s.Language.Valid() {
		return &UserError{Message: "対応していない言語です", Detail: string(s.Language)}
	}
	if strings.TrimSpace(s.OutputDir) == "" {
		return &UserError{Message: "保存先を指定してください", Detail: "empty output dir"}
	}
	if err := ValidateFormats(s.ExportFormats); err != nil {
		return err
	}
	return ValidatePrompt(s.InitialPrompt)
}
