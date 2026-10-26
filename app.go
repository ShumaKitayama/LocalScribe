package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/application"
	"github.com/ShumaKitayama/localscribe/internal/audio"
	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/engine/whisper"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
	"github.com/ShumaKitayama/localscribe/internal/storage"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App は Wails から application.Service を呼び出す。
type App struct {
	ctx      context.Context
	home     string
	svc      *application.Service
	watch    map[string]struct{}
	watchMu  sync.Mutex
	lastEmit map[string]string
	emitMu   sync.Mutex
}

// SaveRecordingResult は Go 側へ保存した録音ファイルの情報である。
type SaveRecordingResult struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

// NewApp は Wails 用の構造体を作る。サービスは startup で初期化する。
func NewApp() *App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return &App{
		home:     home,
		watch:    make(map[string]struct{}),
		lastEmit: make(map[string]string),
	}
}

func (a *App) ensureService() error {
	if a.svc != nil {
		return nil
	}
	if a.home == "" {
		return &domain.UserError{Message: "ホームディレクトリがありません", Detail: "empty home"}
	}
	svc, err := newProductionService(a.home)
	if err != nil {
		return err
	}
	a.svc = svc
	return nil
}

func newProductionService(home string) (*application.Service, error) {
	prober, err := audio.NewProber(audio.ExecRunner{})
	if err != nil {
		return nil, err
	}
	converter, err := audio.NewConverter(audio.ExecRunner{})
	if err != nil {
		return nil, err
	}
	engine, err := whisper.NewEngine(nil)
	if err != nil {
		return nil, err
	}
	return application.NewService(application.ServiceConfig{
		Home:      home,
		Prober:    prober,
		Converter: converter,
		Engine:    engine,
	})
}

// startup は Wails 起動時に呼ばれる。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.ensureService(); err != nil {
		runtime.LogError(ctx, err.Error())
	}
	go a.progressLoop(ctx)
}

// shutdown は Wails 終了時に呼ばれる。
func (a *App) shutdown(context.Context) {
	if a.svc != nil {
		a.svc.Close()
	}
}

// JobSourcePath は再生ハンドラ向けにジョブの元ファイルパスを返す。
func (a *App) JobSourcePath(jobID string) (string, error) {
	if err := a.ensureService(); err != nil {
		return "", err
	}
	job, err := a.svc.GetJob(jobID)
	if err != nil {
		return "", err
	}
	return job.Request.SourcePath, nil
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

// StartTranscription はジョブをキューへ入れ、ID を返す。
func (a *App) StartTranscription(req domain.TranscriptionRequest) (string, error) {
	if err := a.ensureService(); err != nil {
		return "", err
	}
	if req.SourceKind == "" {
		req.SourceKind = domain.SourceFile
	}
	jobID, err := a.svc.StartTranscription(req)
	if err != nil {
		return "", err
	}
	a.trackJob(jobID)
	return jobID, nil
}

// GetJob は保存済みジョブを返す。
func (a *App) GetJob(jobID string) (domain.TranscriptionJob, error) {
	if err := a.ensureService(); err != nil {
		return domain.TranscriptionJob{}, err
	}
	return a.svc.GetJob(jobID)
}

// CancelJob は待機中または実行中のジョブをキャンセルする。
func (a *App) CancelJob(jobID string) error {
	if err := a.ensureService(); err != nil {
		return err
	}
	return a.svc.CancelJob(jobID)
}

// ListRecoverableJobs は interrupted だけを返す。
func (a *App) ListRecoverableJobs() ([]domain.TranscriptionJob, error) {
	if err := a.ensureService(); err != nil {
		return nil, err
	}
	return a.svc.ListRecoverableJobs()
}

// GetTranscript は結果とセグメントを返す。
func (a *App) GetTranscript(transcriptID string) (domain.Transcript, error) {
	if err := a.ensureService(); err != nil {
		return domain.Transcript{}, err
	}
	return a.svc.GetTranscript(transcriptID)
}

// UpdateTranscript は edited_text を保存し、TXT / SRT を再生成する。
func (a *App) UpdateTranscript(transcriptID string, edits []domain.SegmentEdit) error {
	if err := a.ensureService(); err != nil {
		return err
	}
	return a.svc.UpdateTranscript(transcriptID, edits)
}

// ExportTranscript は指定形式を再出力する。
func (a *App) ExportTranscript(transcriptID string, formats []domain.ExportFormat) error {
	if err := a.ensureService(); err != nil {
		return err
	}
	return a.svc.ExportTranscript(transcriptID, formats)
}

// ListModels はカタログとインストール状態を返す。
func (a *App) ListModels() ([]domain.ModelInfo, error) {
	if err := a.ensureService(); err != nil {
		return nil, err
	}
	return a.svc.ListModels()
}

// InstallModel はモデルを取得する。
func (a *App) InstallModel(modelID string) error {
	if err := a.ensureService(); err != nil {
		return err
	}
	return a.svc.InstallModel(a.ctx, modelID)
}

// GetSettings は settings.json を読む。
func (a *App) GetSettings() (domain.AppSettings, error) {
	if err := a.ensureService(); err != nil {
		return domain.AppSettings{}, err
	}
	return a.svc.GetSettings()
}

// UpdateSettings は設定を保存する。
func (a *App) UpdateSettings(settings domain.AppSettings) error {
	if err := a.ensureService(); err != nil {
		return err
	}
	return a.svc.UpdateSettings(settings)
}

// Doctor は環境診断を返す。
func (a *App) Doctor() (application.DoctorReport, error) {
	if err := a.ensureService(); err != nil {
		return application.DoctorReport{}, err
	}
	return a.svc.Doctor()
}

// PickAudioFiles は音声ファイルを複数選択する。
func (a *App) PickAudioFiles() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "音声ファイルを選択",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "音声",
				Pattern:     "*.mp3;*.m4a;*.wav;*.flac;*.mp4;*.mov",
			},
		},
	})
}

// PickOutputDirectory は文字起こしの保存先を選ぶ。
func (a *App) PickOutputDirectory() (string, error) {
	if err := a.ensureService(); err != nil {
		return "", err
	}
	settings, err := a.svc.GetSettings()
	if err != nil {
		return "", err
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "保存先フォルダを選択",
		DefaultDirectory: settings.OutputDir,
	})
}

// RevealOutputFolder は Finder で出力フォルダを開く。
func (a *App) RevealOutputFolder(dir string) error {
	if dir == "" {
		return &domain.UserError{Message: "出力フォルダがありません", Detail: "empty dir"}
	}
	cmd := exec.CommandContext(a.ctx, "open", dir)
	if err := cmd.Run(); err != nil {
		return &domain.UserError{Message: "Finder を開けません", Detail: err.Error()}
	}
	return nil
}

// JobAudioURL はアプリ内再生用の相対 URL を返す。
func (a *App) JobAudioURL(jobID string) string {
	return jobAudioPrefix + jobID
}

func (a *App) trackJob(jobID string) {
	a.watchMu.Lock()
	a.watch[jobID] = struct{}{}
	a.watchMu.Unlock()
}

func (a *App) untrackJob(jobID string) {
	a.watchMu.Lock()
	delete(a.watch, jobID)
	a.watchMu.Unlock()
	a.emitMu.Lock()
	delete(a.lastEmit, jobID)
	a.emitMu.Unlock()
}

func (a *App) progressLoop(ctx context.Context) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.emitJobProgress()
		}
	}
}

func (a *App) emitJobProgress() {
	if a.ctx == nil || a.svc == nil {
		return
	}
	a.watchMu.Lock()
	ids := make([]string, 0, len(a.watch))
	for id := range a.watch {
		ids = append(ids, id)
	}
	a.watchMu.Unlock()

	for _, id := range ids {
		job, err := a.svc.GetJob(id)
		if err != nil {
			a.untrackJob(id)
			continue
		}
		if !job.StartedAt.IsZero() {
			elapsed := time.Since(job.StartedAt).Milliseconds()
			if elapsed < 0 {
				elapsed = 0
			}
			job.Progress.ElapsedMS = elapsed
		}
		job.Progress.JobID = job.ID
		job.Progress.State = job.State
		key := jobProgressKey(job)
		a.emitMu.Lock()
		if a.lastEmit[id] == key {
			a.emitMu.Unlock()
			if job.State.Terminal() {
				a.untrackJob(id)
			}
			continue
		}
		a.lastEmit[id] = key
		a.emitMu.Unlock()
		runtime.EventsEmit(a.ctx, "job:progress", job)
		if job.State.Terminal() {
			a.untrackJob(id)
		}
	}
}

func jobProgressKey(job domain.TranscriptionJob) string {
	pct := "nil"
	if job.Progress.Percent != nil {
		pct = fmt.Sprintf("%f", *job.Progress.Percent)
	}
	return fmt.Sprintf("%s|%s|%s|%s|%s", job.State, job.Progress.Message, pct, job.TranscriptID, job.ResultDir)
}
