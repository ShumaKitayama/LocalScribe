package domain

import (
	"strings"
	"time"
)

// TranscriptionRequest は 1 回の文字起こしに必要な入力である。
type TranscriptionRequest struct {
	SourcePath    string         `json:"source_path"`
	SourceKind    SourceKind     `json:"source_kind,omitempty"`
	ModelID       string         `json:"model_id"`
	Language      Language       `json:"language"`
	OutputDir     string         `json:"output_dir"`
	InitialPrompt string         `json:"initial_prompt"`
	ExportFormats []ExportFormat `json:"export_formats"`
}

// Validate はジョブを登録する前の入力条件を確認する。ファイルの中身は読まない。
func (r TranscriptionRequest) Validate() error {
	if strings.TrimSpace(r.SourcePath) == "" {
		return &UserError{Message: "入力ファイルを指定してください", Detail: "empty source path"}
	}
	if !r.SourceKind.Valid() {
		return &UserError{Message: "対応していない入力種別です", Detail: string(r.SourceKind)}
	}
	if r.SourceKind == SourceRecording {
		if !AllowedRecordingExtension(r.SourcePath) {
			return &UserError{Message: "録音ファイルの形式が対応外です", Detail: r.SourcePath}
		}
	} else if !AllowedUserExtension(r.SourcePath) {
		return &UserError{Message: "対応していないファイル形式です", Detail: r.SourcePath}
	}
	if !ValidModelID(r.ModelID) {
		return &UserError{Message: "対応していないモデルです", Detail: r.ModelID}
	}
	if !r.Language.Valid() {
		return &UserError{Message: "対応していない言語です", Detail: string(r.Language)}
	}
	if strings.TrimSpace(r.OutputDir) == "" {
		return &UserError{Message: "保存先を指定してください", Detail: "empty output dir"}
	}
	if err := ValidateFormats(r.ExportFormats); err != nil {
		return err
	}
	return ValidatePrompt(r.InitialPrompt)
}

// Transcript は文字起こし結果の正本になるデータである。
type Transcript struct {
	SchemaVersion int       `json:"schema_version"`
	ID            string    `json:"id"`
	JobID         string    `json:"job_id"`
	ModelID       string    `json:"model_id"`
	Language      Language  `json:"language"`
	SourcePath    string    `json:"source_path"`
	DurationMS    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
	Segments      []Segment `json:"segments"`
}

// Validate は結果を保存してよいかを判定する。
func (t Transcript) Validate() error {
	if t.SchemaVersion != SchemaVersion {
		return &UserError{Message: "文字起こし結果の形式が対応外です", Detail: "schema_version"}
	}
	if t.ID == "" || t.JobID == "" {
		return &UserError{Message: "文字起こし結果のIDがありません", Detail: "missing id"}
	}
	if !ValidModelID(t.ModelID) {
		return &UserError{Message: "対応していないモデルです", Detail: t.ModelID}
	}
	if !t.Language.Valid() {
		return &UserError{Message: "対応していない言語です", Detail: string(t.Language)}
	}
	if t.CreatedAt.IsZero() {
		return &UserError{Message: "作成時刻がありません", Detail: "created_at"}
	}
	if t.DurationMS < 0 {
		return &UserError{Message: "音声の長さが不正です", Detail: "duration_ms"}
	}
	for i := range t.Segments {
		if err := t.Segments[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ApplyEdits は範囲をすべて確認してから編集後テキストだけを更新する。
func (t *Transcript) ApplyEdits(edits []SegmentEdit) error {
	if t == nil {
		return &UserError{Message: "文字起こし結果がありません", Detail: "nil transcript"}
	}
	for _, edit := range edits {
		if edit.Index < 0 || edit.Index >= len(t.Segments) {
			return &UserError{Message: "修正する文が見つかりません", Detail: "segment index out of range"}
		}
	}
	for _, edit := range edits {
		t.Segments[edit.Index].ApplyEdit(edit.EditedText)
	}
	return nil
}
