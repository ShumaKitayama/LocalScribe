// Package domain は文字起こしの型と不変条件を持つ。ファイル操作とプロセス起動はしない。
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"slices"
	"time"
)

// SchemaVersion は保存する JSON の版である。
const SchemaVersion = 1

// JobState はジョブの状態名である。保存値は設計書の英小文字に固定する。
type JobState string

const (
	StateQueued       JobState = "queued"
	StateValidating   JobState = "validating"
	StateConverting   JobState = "converting"
	StateTranscribing JobState = "transcribing"
	StateExporting    JobState = "exporting"
	StateCompleted    JobState = "completed"
	StateFailed       JobState = "failed"
	StateCancelled    JobState = "cancelled"
	StateInterrupted  JobState = "interrupted"
)

var (
	activeStates = []JobState{
		StateQueued,
		StateValidating,
		StateConverting,
		StateTranscribing,
		StateExporting,
	}
	terminalStates = []JobState{
		StateCompleted,
		StateFailed,
		StateCancelled,
		StateInterrupted,
	}
)

// Valid は定義済みの状態なら true を返す。
func (s JobState) Valid() bool {
	return slices.Contains(activeStates, s) || slices.Contains(terminalStates, s)
}

// Terminal は、再実行せず新しいジョブを作る終了状態なら true を返す。
func (s JobState) Terminal() bool {
	return slices.Contains(terminalStates, s)
}

// JobProgress は処理の段階と経過を表す。Percent が nil のときは未計測であり、0 を入れない。
type JobProgress struct {
	JobID     string   `json:"job_id"`
	State     JobState `json:"state"`
	ElapsedMS int64    `json:"elapsed_ms"`
	Message   string   `json:"message"`
	Percent   *float64 `json:"percent,omitempty"`
}

// TranscriptionJob は 1 回の文字起こし作業である。
type TranscriptionJob struct {
	SchemaVersion int                  `json:"schema_version"`
	ID            string               `json:"id"`
	State         JobState             `json:"state"`
	Request       TranscriptionRequest `json:"request"`
	StartedAt     time.Time            `json:"started_at,omitzero"`
	EndedAt       time.Time            `json:"ended_at,omitzero"`
	Error         *UserError           `json:"error,omitempty"`
	TranscriptID  string               `json:"transcript_id,omitempty"`
	ResultDir     string               `json:"result_dir,omitempty"`
	Progress      JobProgress          `json:"progress"`
}

// ErrTerminalJob は、終了したジョブを別の状態へ動かそうとしたときに返す。
var ErrTerminalJob = &UserError{
	Message: "終了したジョブは再開できません",
	Detail:  "terminal job cannot change state",
}

// NewTranscriptionJob は待機状態のジョブを作る。入力が不正ならジョブは作らない。
func NewTranscriptionJob(id string, req TranscriptionRequest) (TranscriptionJob, error) {
	if id == "" {
		return TranscriptionJob{}, &UserError{Message: "ジョブIDがありません", Detail: "empty id"}
	}
	if err := req.Validate(); err != nil {
		return TranscriptionJob{}, err
	}
	return TranscriptionJob{
		SchemaVersion: SchemaVersion,
		ID:            id,
		State:         StateQueued,
		Request:       req,
		Progress: JobProgress{
			JobID:   id,
			State:   StateQueued,
			Message: "待機しています",
		},
	}, nil
}

// Transition は次の状態へ進める。終了状態から動かすことはできない。
func (j *TranscriptionJob) Transition(next JobState) error {
	if j == nil {
		return &UserError{Message: "ジョブがありません", Detail: "nil job"}
	}
	if j.State.Terminal() {
		return ErrTerminalJob
	}
	if !next.Valid() {
		return &UserError{Message: "対応していない処理状態です", Detail: string(next)}
	}
	j.State = next
	j.Progress.JobID = j.ID
	j.Progress.State = next
	return nil
}

// RandomID は 16 バイトを 32 文字の hex にした ID を返す。
func RandomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", &UserError{Message: "IDを発行できません", Detail: err.Error()}
	}
	return hex.EncodeToString(buf[:]), nil
}

// NewID は gen が nil なら RandomID を使う。テストは gen を渡して ID を固定できる。
func NewID(gen func() (string, error)) (string, error) {
	if gen == nil {
		return RandomID()
	}
	return gen()
}
