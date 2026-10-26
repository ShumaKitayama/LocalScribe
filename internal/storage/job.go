package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
)

const interruptedMessage = "前回の処理が完了していません"

// JobStore は 1 ジョブ 1 JSON の保存を行う。
type JobStore struct {
	dir string
}

// NewJobStore は jobs ディレクトリを用意してストアを作る。
func NewJobStore(home string) (JobStore, error) {
	if err := validateHome(home); err != nil {
		return JobStore{}, err
	}
	dir := macos.JobsDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return JobStore{}, fmt.Errorf("create jobs dir: %w", err)
	}
	return JobStore{dir: dir}, nil
}

// NewJobStoreDir はテスト用に jobs ディレクトリを直接指定する。
func NewJobStoreDir(dir string) (JobStore, error) {
	if strings.TrimSpace(dir) == "" {
		return JobStore{}, &domain.UserError{Message: "ジョブの保存先がありません", Detail: "empty jobs dir"}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return JobStore{}, fmt.Errorf("create jobs dir: %w", err)
	}
	return JobStore{dir: dir}, nil
}

// Save はジョブを jobs/<id>.json に原子的に書く。
func (s JobStore) Save(job domain.TranscriptionJob) error {
	if job.ID == "" {
		return &domain.UserError{Message: "ジョブIDがありません", Detail: "empty id"}
	}
	if err := jobPathSafe(job.ID); err != nil {
		return err
	}
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return fmt.Errorf("encode job %s: %w", job.ID, err)
	}
	data = append(data, '\n')
	path := s.jobPath(job.ID)
	if err := WriteFileAtomic(path, data); err != nil {
		return err
	}
	return nil
}

// Load は 1 件のジョブを読む。
func (s JobStore) Load(id string) (domain.TranscriptionJob, error) {
	if err := jobPathSafe(id); err != nil {
		return domain.TranscriptionJob{}, err
	}
	data, err := os.ReadFile(s.jobPath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.TranscriptionJob{}, &domain.UserError{Message: "ジョブが見つかりません", Detail: id}
		}
		return domain.TranscriptionJob{}, fmt.Errorf("read job %s: %w", id, err)
	}
	var job domain.TranscriptionJob
	if err := json.Unmarshal(data, &job); err != nil {
		return domain.TranscriptionJob{}, fmt.Errorf("decode job %s: %w", id, err)
	}
	return job, nil
}

// List は jobs 配下の JSON をすべて読む。ファイル名の順で返す。
func (s JobStore) List() ([]domain.TranscriptionJob, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read jobs dir: %w", err)
	}
	var jobs []domain.TranscriptionJob
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		job, err := s.Load(id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// MarkInterruptedOnStartup は終了していないジョブを interrupted に更新する。自動再開はしない。
func (s JobStore) MarkInterruptedOnStartup(now time.Time) error {
	jobs, err := s.List()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.State.Terminal() {
			continue
		}
		job.State = domain.StateInterrupted
		job.Progress.State = domain.StateInterrupted
		job.Progress.Message = interruptedMessage
		if job.EndedAt.IsZero() {
			job.EndedAt = now.UTC()
		}
		if err := s.Save(job); err != nil {
			return err
		}
	}
	return nil
}

func (s JobStore) jobPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func jobPathSafe(id string) error {
	if strings.Contains(id, "/") || strings.Contains(id, "\\") || id == "." || id == ".." {
		return &domain.UserError{Message: "ジョブIDが不正です", Detail: id}
	}
	return nil
}
