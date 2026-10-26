package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/audio"
	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/engine"
	"github.com/ShumaKitayama/localscribe/internal/export"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
	"github.com/ShumaKitayama/localscribe/internal/storage"
)

// AudioProber は入力の長さと音声ストリームを調べる。
type AudioProber interface {
	Probe(ctx context.Context, inputPath string) (audio.ProbeInfo, error)
}

// AudioConverter は作業ディレクトリへ 16 kHz WAV を書く。
type AudioConverter interface {
	Convert(ctx context.Context, workDir, inputPath, outputWAV string, probe audio.ProbeInfo) error
}

// DoctorReport は環境診断の結果である。
type DoctorReport struct {
	Issues []string `json:"issues"`
}

// Service は文字起こしの公開 API を提供する。
type Service struct {
	home      string
	workDir   string
	jobs      storage.JobStore
	settings  storage.SettingsStore
	prober    AudioProber
	converter AudioConverter
	engine    engine.Engine
	models    *ModelService
	manager   *JobManager
	now       func() time.Time
	idGen     func() (string, error)

	workerCtx    context.Context
	workerCancel context.CancelFunc
}

// ServiceConfig は依存を差し替えるための設定である。
type ServiceConfig struct {
	Home         string
	JobsDir      string
	Prober       AudioProber
	Converter    AudioConverter
	Engine       engine.Engine
	ModelHTTP    HTTPDoer
	ModelCatalog []CatalogEntry
	Now          func() time.Time
	IDGen        func() (string, error)
}

// NewService は保存層を初期化し、中断マーク後にワーカーを 1 本だけ起動する。
func NewService(cfg ServiceConfig) (*Service, error) {
	if strings.TrimSpace(cfg.Home) == "" {
		return nil, &domain.UserError{Message: "ホームディレクトリがありません", Detail: "empty home"}
	}
	if cfg.Prober == nil || cfg.Converter == nil || cfg.Engine == nil {
		return nil, &domain.UserError{Message: "音声処理の依存が不足しています", Detail: "missing deps"}
	}
	var jobs storage.JobStore
	var err error
	if cfg.JobsDir != "" {
		jobs, err = storage.NewJobStoreDir(cfg.JobsDir)
	} else {
		jobs, err = storage.NewJobStore(cfg.Home)
	}
	if err != nil {
		return nil, err
	}
	settings, err := storage.NewSettingsStore(cfg.Home)
	if err != nil {
		return nil, err
	}
	workDir := macos.WorkCacheDir(cfg.Home)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, err
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	idGen := cfg.IDGen
	if idGen == nil {
		idGen = domain.RandomID
	}
	if err := jobs.MarkInterruptedOnStartup(now().UTC()); err != nil {
		return nil, err
	}
	models, err := NewModelService(ModelServiceConfig{
		Home:    cfg.Home,
		Catalog: cfg.ModelCatalog,
		HTTP:    cfg.ModelHTTP,
	})
	if err != nil {
		return nil, err
	}
	workerCtx, workerCancel := context.WithCancel(context.Background())
	svc := &Service{
		home:         cfg.Home,
		workDir:      workDir,
		jobs:         jobs,
		settings:     settings,
		prober:       cfg.Prober,
		converter:    cfg.Converter,
		engine:       cfg.Engine,
		models:       models,
		manager:      NewJobManager(),
		now:          now,
		idGen:        idGen,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}
	go svc.runWorker(workerCtx)
	return svc, nil
}

// Close はワーカーを止める。
func (s *Service) Close() {
	s.workerCancel()
}

// StartTranscription はジョブを queued で保存し、ID をすぐ返す。
func (s *Service) StartTranscription(req domain.TranscriptionRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}
	if err := ValidateModelAvailable(s.home, req.ModelID); err != nil {
		return "", err
	}
	if _, err := os.Stat(req.SourcePath); err != nil {
		return "", &domain.UserError{Message: "入力ファイルを開けません", Detail: err.Error()}
	}
	jobID, err := domain.NewID(s.idGen)
	if err != nil {
		return "", err
	}
	job, err := domain.NewTranscriptionJob(jobID, req)
	if err != nil {
		return "", err
	}
	if err := s.jobs.Save(job); err != nil {
		return "", err
	}
	s.manager.Enqueue(jobID)
	return jobID, nil
}

// GetJob は保存済みジョブを返す。
func (s *Service) GetJob(jobID string) (domain.TranscriptionJob, error) {
	return s.jobs.Load(jobID)
}

// CancelJob は待機中または実行中のジョブをキャンセルする。
func (s *Service) CancelJob(jobID string) error {
	job, err := s.jobs.Load(jobID)
	if err != nil {
		return err
	}
	if job.State.Terminal() {
		return &domain.UserError{Message: "終了したジョブはキャンセルできません", Detail: jobID}
	}
	if job.State == domain.StateQueued {
		s.manager.RemoveQueued(jobID)
		return s.finishCancelled(&job)
	}
	if s.manager.CancelActive(jobID) {
		return nil
	}
	return &domain.UserError{Message: "ジョブをキャンセルできません", Detail: jobID}
}

// ListRecoverableJobs は interrupted だけを返す。
func (s *Service) ListRecoverableJobs() ([]domain.TranscriptionJob, error) {
	jobs, err := s.jobs.List()
	if err != nil {
		return nil, err
	}
	var out []domain.TranscriptionJob
	for _, job := range jobs {
		if job.State == domain.StateInterrupted {
			out = append(out, job)
		}
	}
	return out, nil
}

// GetTranscript は結果ディレクトリから正本 JSON を読む。
func (s *Service) GetTranscript(transcriptID string) (domain.Transcript, error) {
	dir, err := s.findResultDir(transcriptID)
	if err != nil {
		return domain.Transcript{}, err
	}
	return storage.LoadTranscript(dir)
}

// UpdateTranscript は edited_text を保存し、TXT と SRT を JSON から再生成する。
func (s *Service) UpdateTranscript(transcriptID string, edits []domain.SegmentEdit) error {
	dir, err := s.findResultDir(transcriptID)
	if err != nil {
		return err
	}
	transcript, err := storage.LoadTranscript(dir)
	if err != nil {
		return err
	}
	if err := transcript.ApplyEdits(edits); err != nil {
		return err
	}
	if err := storage.SaveTranscript(dir, transcript); err != nil {
		return err
	}
	return writeDerivedExports(dir, transcript)
}

// ExportTranscript は JSON 正本から指定形式を再出力する。
func (s *Service) ExportTranscript(transcriptID string, formats []domain.ExportFormat) error {
	if err := domain.ValidateFormats(formats); err != nil {
		return err
	}
	dir, err := s.findResultDir(transcriptID)
	if err != nil {
		return err
	}
	transcript, err := storage.LoadTranscript(dir)
	if err != nil {
		return err
	}
	return writeExports(dir, transcript, formats)
}

// ListModels はカタログとファイル状態を返す。
func (s *Service) ListModels() ([]domain.ModelInfo, error) {
	return s.models.List()
}

// InstallModel はカタログからモデルを取得する。
func (s *Service) InstallModel(ctx context.Context, modelID string) error {
	return s.models.Install(ctx, modelID)
}

// GetSettings は settings.json を読む。
func (s *Service) GetSettings() (domain.AppSettings, error) {
	return s.settings.Load()
}

// UpdateSettings は設定を保存する。
func (s *Service) UpdateSettings(settings domain.AppSettings) error {
	return s.settings.Save(settings)
}

// Doctor は簡易な環境診断を返す。
func (s *Service) Doctor() (DoctorReport, error) {
	var issues []string
	if err := ValidateModelAvailable(s.home, domain.ModelLargeV3); err != nil {
		issues = append(issues, err.Error())
	}
	return DoctorReport{Issues: issues}, nil
}

func (s *Service) runWorker(ctx context.Context) {
	for {
		jobID, ok := s.manager.Dequeue(ctx)
		if !ok {
			return
		}
		job, err := s.jobs.Load(jobID)
		if err != nil {
			continue
		}
		if job.State != domain.StateQueued {
			continue
		}
		runCtx, cancel := context.WithCancel(context.Background())
		s.manager.SetActive(jobID, cancel)
		s.processJob(runCtx, job)
		s.manager.ClearActive()
		cancel()
	}
}

func (s *Service) processJob(ctx context.Context, job domain.TranscriptionJob) {
	started := s.now().UTC()
	job.StartedAt = started
	if err := s.setState(&job, domain.StateValidating, "入力を確認しています"); err != nil {
		return
	}

	probe, err := s.prober.Probe(ctx, job.Request.SourcePath)
	if err != nil {
		s.finishFailed(&job, err)
		return
	}
	if ctx.Err() != nil {
		s.finishCancelled(&job)
		return
	}

	if err := s.setState(&job, domain.StateConverting, "音声を変換しています"); err != nil {
		return
	}
	workWAV := filepath.Join(s.workDir, job.ID+".wav")
	if err := s.converter.Convert(ctx, s.workDir, job.Request.SourcePath, workWAV, probe); err != nil {
		if engine.IsCancelled(err) || errors.Is(err, context.Canceled) {
			s.finishCancelled(&job)
		} else {
			s.finishFailed(&job, err)
		}
		s.cleanupWork(job.ID)
		return
	}
	if ctx.Err() != nil {
		s.finishCancelled(&job)
		s.cleanupWork(job.ID)
		return
	}

	if err := s.setState(&job, domain.StateTranscribing, "音声を認識しています"); err != nil {
		s.cleanupWork(job.ID)
		return
	}
	prefix := filepath.Join(s.workDir, job.ID+"-whisper-out")
	segments, err := s.engine.Transcribe(ctx, engine.Request{
		WAVPath:       workWAV,
		ModelPath:     modelPath(s.home, job.Request.ModelID),
		Language:      job.Request.Language,
		InitialPrompt: job.Request.InitialPrompt,
		WorkDir:       s.workDir,
		OutputPrefix:  prefix,
	}, func(p engine.Progress) {
		_ = s.updateProgress(&job, domain.StateTranscribing, p.Message, p.Percent)
	})
	if err != nil {
		if engine.IsCancelled(err) || errors.Is(err, context.Canceled) {
			s.finishCancelled(&job)
		} else {
			s.finishFailed(&job, err)
		}
		s.cleanupWork(job.ID)
		return
	}

	if err := s.setState(&job, domain.StateExporting, "結果を保存しています"); err != nil {
		s.cleanupWork(job.ID)
		return
	}
	transcriptID, err := domain.NewID(s.idGen)
	if err != nil {
		s.finishFailed(&job, err)
		s.cleanupWork(job.ID)
		return
	}
	transcript := domain.Transcript{
		SchemaVersion: domain.SchemaVersion,
		ID:            transcriptID,
		JobID:         job.ID,
		ModelID:       job.Request.ModelID,
		Language:      job.Request.Language,
		SourcePath:    job.Request.SourcePath,
		DurationMS:    probe.DurationMS,
		CreatedAt:     s.now().UTC(),
		Segments:      segments,
	}
	outDir, err := storage.PrepareOutputDir(job.Request.OutputDir, job.Request.SourcePath, s.now())
	if err != nil {
		s.finishFailed(&job, err)
		s.cleanupWork(job.ID)
		return
	}
	if err := storage.SaveTranscript(outDir, transcript); err != nil {
		s.finishFailed(&job, err)
		s.cleanupWork(job.ID)
		return
	}
	if err := writeExports(outDir, transcript, job.Request.ExportFormats); err != nil {
		s.finishFailed(&job, err)
		s.cleanupWork(job.ID)
		return
	}

	job.TranscriptID = transcript.ID
	job.ResultDir = outDir
	job.EndedAt = s.now().UTC()
	if err := s.setState(&job, domain.StateCompleted, "完了しました"); err != nil {
		s.cleanupWork(job.ID)
		return
	}
	s.cleanupWork(job.ID)
}

func (s *Service) setState(job *domain.TranscriptionJob, state domain.JobState, message string) error {
	if err := job.Transition(state); err != nil {
		s.finishFailed(job, err)
		return err
	}
	job.Progress.Message = message
	return s.jobs.Save(*job)
}

func (s *Service) updateProgress(job *domain.TranscriptionJob, state domain.JobState, message string, percent *float64) error {
	job.Progress.State = state
	job.Progress.Message = message
	job.Progress.Percent = percent
	return s.jobs.Save(*job)
}

func (s *Service) finishFailed(job *domain.TranscriptionJob, err error) {
	userErr := asUserError(err)
	_ = job.Transition(domain.StateFailed)
	job.Error = userErr
	job.EndedAt = s.now().UTC()
	job.Progress.State = domain.StateFailed
	job.Progress.Message = userErr.Message
	_ = s.jobs.Save(*job)
}

func (s *Service) finishCancelled(job *domain.TranscriptionJob) error {
	if job.State.Terminal() {
		return nil
	}
	_ = job.Transition(domain.StateCancelled)
	job.EndedAt = s.now().UTC()
	job.Progress.State = domain.StateCancelled
	job.Progress.Message = "キャンセルしました"
	return s.jobs.Save(*job)
}

func (s *Service) cleanupWork(jobID string) {
	names := []string{
		jobID + ".wav",
		jobID + "-whisper-out.json",
	}
	for _, name := range names {
		_ = os.Remove(filepath.Join(s.workDir, name))
	}
}

func (s *Service) findResultDir(transcriptID string) (string, error) {
	jobs, err := s.jobs.List()
	if err != nil {
		return "", err
	}
	for _, job := range jobs {
		if job.TranscriptID == transcriptID && job.ResultDir != "" {
			return job.ResultDir, nil
		}
	}
	return "", &domain.UserError{Message: "文字起こし結果が見つかりません", Detail: transcriptID}
}

func writeExports(dir string, transcript domain.Transcript, formats []domain.ExportFormat) error {
	for _, format := range formats {
		switch format {
		case domain.FormatJSON:
			data, err := export.EncodeTranscriptJSON(transcript)
			if err != nil {
				return err
			}
			if err := storage.WriteFileAtomic(filepath.Join(dir, "transcript.json"), data); err != nil {
				return err
			}
		case domain.FormatTXT:
			content := export.RenderTXT(transcript)
			if err := storage.WriteFileAtomic(filepath.Join(dir, "transcript.txt"), []byte(content)); err != nil {
				return err
			}
		case domain.FormatSRT:
			content := export.RenderSRT(transcript)
			if err := storage.WriteFileAtomic(filepath.Join(dir, "transcript.srt"), []byte(content)); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeDerivedExports(dir string, transcript domain.Transcript) error {
	return writeExports(dir, transcript, []domain.ExportFormat{
		domain.FormatTXT,
		domain.FormatSRT,
	})
}

func modelPath(home, modelID string) string {
	fileName, _ := domain.ModelFileName(modelID)
	return filepath.Join(macos.ModelsDir(home), fileName)
}

func asUserError(err error) *domain.UserError {
	if err == nil {
		return &domain.UserError{Message: "処理に失敗しました", Detail: "unknown"}
	}
	var userErr *domain.UserError
	if errors.As(err, &userErr) {
		return userErr
	}
	return &domain.UserError{Message: "処理に失敗しました", Detail: err.Error()}
}
