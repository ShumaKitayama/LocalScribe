package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ShumaKitayama/localscribe/internal/application"
	"github.com/ShumaKitayama/localscribe/internal/audio"
	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/engine/whisper"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
	"github.com/ShumaKitayama/localscribe/internal/storage"
)

const (
	exitOK       = 0
	exitUsage    = 1
	exitRuntime  = 2
	exitCanceled = 3
)

// AppService は CLI が使うアプリケーション API である。
type AppService interface {
	StartTranscription(req domain.TranscriptionRequest) (string, error)
	GetJob(jobID string) (domain.TranscriptionJob, error)
	CancelJob(jobID string) error
	ListModels() ([]domain.ModelInfo, error)
	InstallModel(ctx context.Context, modelID string) error
	GetSettings() (domain.AppSettings, error)
	Close()
}

type transcribeOutput struct {
	JobID     string   `json:"job_id"`
	State     string   `json:"state"`
	OutputDir string   `json:"output_dir"`
	Files     []string `json:"files"`
}

// commandBackends は、引数を確認したあとで必要なサービスだけを作る。
type commandBackends struct {
	transcribe func() (AppService, func(), error)
	models     func() (AppService, error)
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitRuntime)
	}
	os.Exit(Run(os.Args, os.Stdout, os.Stderr, productionBackends(home), home))
}

func productionBackends(home string) commandBackends {
	return commandBackends{
		transcribe: func() (AppService, func(), error) {
			svc, err := newProductionService(home)
			if err != nil {
				return nil, nil, err
			}
			return svc, svc.Close, nil
		},
		models: func() (AppService, error) {
			return newModelsCommandService(home)
		},
	}
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

// Run は argv を実行し、終了コードを返す。argv[0] はコマンド名。
func Run(argv []string, stdout, stderr io.Writer, backends commandBackends, home string) int {
	if len(argv) < 2 {
		printUsage(stderr)
		return exitUsage
	}
	switch argv[1] {
	case "transcribe":
		return runTranscribe(argv[2:], stdout, stderr, backends.transcribe, true)
	case "doctor":
		return runDoctor(stdout, home)
	case "models":
		if len(argv) < 3 {
			fmt.Fprintln(stderr, "models には list または install が必要です")
			return exitUsage
		}
		switch argv[2] {
		case "list":
			return runModelsList(stdout, stderr, backends.models)
		case "install":
			if len(argv) < 4 {
				fmt.Fprintln(stderr, "models install にはモデル ID が必要です")
				return exitUsage
			}
			return runModelsInstall(argv[3], stderr, backends.models)
		default:
			fmt.Fprintln(stderr, "未知の models サブコマンドです")
			return exitUsage
		}
	default:
		fmt.Fprintln(stderr, "未知のコマンドです")
		return exitUsage
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  localscribe transcribe <file> --model <id> --language <ja|en|auto> --format txt,srt,json [--output <dir>] [--prompt <text>]")
	fmt.Fprintln(w, "  localscribe models list")
	fmt.Fprintln(w, "  localscribe models install <model-id>")
	fmt.Fprintln(w, "  localscribe doctor")
}

func runTranscribe(args []string, stdout, stderr io.Writer, load func() (AppService, func(), error), handleSignal bool) int {
	input, flagArgs := extractInputPath(args)
	fs := flag.NewFlagSet("transcribe", flag.ContinueOnError)
	fs.SetOutput(stderr)
	model := fs.String("model", "", "model id")
	language := fs.String("language", "", "language")
	format := fs.String("format", "", "export formats")
	output := fs.String("output", "", "output directory")
	prompt := fs.String("prompt", "", "initial prompt")
	if err := fs.Parse(flagArgs); err != nil {
		return exitUsage
	}
	if input == "" || *model == "" || *language == "" || *format == "" {
		fmt.Fprintln(stderr, "入力ファイル、--model、--language、--format が必要です")
		return exitUsage
	}
	lang, err := domain.ParseLanguage(*language)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	formats, err := parseFormats(*format)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	if load == nil {
		fmt.Fprintln(stderr, "文字起こしを開始できません")
		return exitRuntime
	}
	svc, closeSvc, err := load()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitRuntime
	}
	if closeSvc != nil {
		defer closeSvc()
	}
	outputDir := *output
	if strings.TrimSpace(outputDir) == "" {
		settings, err := svc.GetSettings()
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return exitRuntime
		}
		outputDir = settings.OutputDir
	}
	req := domain.TranscriptionRequest{
		SourcePath:    input,
		ModelID:       *model,
		Language:      lang,
		OutputDir:     outputDir,
		InitialPrompt: *prompt,
		ExportFormats: formats,
	}
	jobID, err := svc.StartTranscription(req)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		if isUsageError(err) {
			return exitUsage
		}
		return exitRuntime
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	if handleSignal {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigCh)
		go func() {
			select {
			case <-sigCh:
				_ = svc.CancelJob(jobID)
				stop()
			case <-ctx.Done():
			}
		}()
	}

	lastMessage := ""
	for {
		job, err := svc.GetJob(jobID)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return exitRuntime
		}
		if job.Progress.Message != "" && job.Progress.Message != lastMessage {
			fmt.Fprintln(stderr, job.Progress.Message)
			lastMessage = job.Progress.Message
		}
		if job.State.Terminal() {
			switch job.State {
			case domain.StateCompleted:
				out := transcribeOutput{
					JobID:     job.ID,
					State:     string(job.State),
					OutputDir: job.ResultDir,
					Files:     outputFileNames(job.ResultDir, formats),
				}
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(out); err != nil {
					fmt.Fprintln(stderr, err.Error())
					return exitRuntime
				}
				return exitOK
			case domain.StateCancelled:
				return exitCanceled
			default:
				if job.Error != nil {
					fmt.Fprintln(stderr, job.Error.Message)
				}
				return exitRuntime
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func runModelsList(stdout, stderr io.Writer, load func() (AppService, error)) int {
	if load == nil {
		fmt.Fprintln(stderr, "モデル一覧を取得できません")
		return exitRuntime
	}
	svc, err := load()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitRuntime
	}
	models, err := svc.ListModels()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitRuntime
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(models); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitRuntime
	}
	return exitOK
}

func runModelsInstall(modelID string, stderr io.Writer, load func() (AppService, error)) int {
	if !domain.ValidModelID(modelID) {
		fmt.Fprintln(stderr, "対応していないモデルです")
		return exitUsage
	}
	if load == nil {
		fmt.Fprintln(stderr, "モデルを取得できません")
		return exitRuntime
	}
	svc, err := load()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitRuntime
	}
	if err := svc.InstallModel(context.Background(), modelID); err != nil {
		fmt.Fprintln(stderr, err.Error())
		if isUsageError(err) {
			return exitUsage
		}
		return exitRuntime
	}
	return exitOK
}

func runDoctor(stdout io.Writer, home string) int {
	issues := doctorIssues(home)
	fmt.Fprintln(stdout, "LocalScribe doctor")
	for _, line := range doctorLines(home) {
		fmt.Fprintln(stdout, line)
	}
	if len(issues) > 0 {
		return exitRuntime
	}
	return exitOK
}

func doctorLines(home string) []string {
	var lines []string
	osLine := fmt.Sprintf("OS: %s/%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		osLine += " OK"
	} else {
		osLine += " NG"
	}
	lines = append(lines, osLine)

	ffmpeg, err := audio.ResolveFFmpegPath()
	lines = append(lines, toolLine("FFmpeg", ffmpeg, err))

	ffprobe, err := audio.ResolveFFprobePath()
	lines = append(lines, toolLine("FFprobe", ffprobe, err))

	whisperBin, err := whisper.ResolveBinary()
	lines = append(lines, toolLine("whisper-cli", whisperBin, err))

	models, _ := application.NewModelService(application.ModelServiceConfig{Home: home})
	if models != nil {
		list, err := models.List()
		if err != nil {
			lines = append(lines, "Models: 確認できません")
		} else {
			for _, model := range list {
				status := string(model.Status)
				if model.Status == domain.ModelPartial {
					status = "partial NG"
				} else if model.Status == domain.ModelAvailable {
					status = "available OK"
				}
				lines = append(lines, fmt.Sprintf("Model %s: %s", model.ID, status))
			}
		}
	}

	settingsStore, err := storage.NewSettingsStore(home)
	outputDir := macos.DefaultTranscriptsDir(home)
	if err == nil {
		settings, loadErr := settingsStore.Load()
		if loadErr == nil {
			outputDir = settings.OutputDir
		}
	}
	lines = append(lines, writableLine("Output", outputDir))

	appSupport := macos.ApplicationSupportDir(home)
	workDir := macos.WorkCacheDir(home)
	lines = append(lines, spaceLine("Application Support", appSupport))
	lines = append(lines, spaceLine("Work cache", workDir))
	return lines
}

func doctorIssues(home string) []string {
	var issues []string
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		issues = append(issues, "OS is not darwin/arm64")
	}
	if _, err := audio.ResolveFFmpegPath(); err != nil {
		issues = append(issues, err.Error())
	}
	if _, err := audio.ResolveFFprobePath(); err != nil {
		issues = append(issues, err.Error())
	}
	if _, err := whisper.ResolveBinary(); err != nil {
		issues = append(issues, err.Error())
	}
	models, err := application.NewModelService(application.ModelServiceConfig{Home: home})
	if err == nil {
		list, listErr := models.List()
		if listErr != nil {
			issues = append(issues, listErr.Error())
		} else {
			for _, model := range list {
				if model.Status == domain.ModelPartial {
					issues = append(issues, fmt.Sprintf("model %s is partial", model.ID))
				}
			}
		}
	}
	settingsStore, err := storage.NewSettingsStore(home)
	outputDir := macos.DefaultTranscriptsDir(home)
	if err == nil {
		settings, loadErr := settingsStore.Load()
		if loadErr == nil {
			outputDir = settings.OutputDir
		}
	}
	if !dirWritable(outputDir) {
		issues = append(issues, "output directory is not writable")
	}
	_ = os.MkdirAll(macos.ApplicationSupportDir(home), 0o755)
	if _, err := audio.AvailableBytes(macos.ApplicationSupportDir(home)); err != nil {
		issues = append(issues, "cannot verify Application Support free space")
	}
	_ = os.MkdirAll(macos.WorkCacheDir(home), 0o755)
	if _, err := audio.AvailableBytes(macos.WorkCacheDir(home)); err != nil {
		issues = append(issues, "cannot verify work cache free space")
	}
	return issues
}

func toolLine(name, path string, err error) string {
	if err != nil {
		return fmt.Sprintf("%s: 見つかりません", name)
	}
	return fmt.Sprintf("%s: %s OK", name, path)
}

func writableLine(label, dir string) string {
	if dirWritable(dir) {
		return fmt.Sprintf("%s: %s OK", label, dir)
	}
	return fmt.Sprintf("%s: %s NG", label, dir)
}

func spaceLine(label, dir string) string {
	free, err := audio.AvailableBytes(dir)
	if err != nil {
		return fmt.Sprintf("%s: %s (空き容量不明)", label, dir)
	}
	return fmt.Sprintf("%s: %s (空き %d bytes)", label, dir, free)
}

func dirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	test := filepath.Join(dir, ".write-test")
	if err := os.WriteFile(test, []byte("1"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(test)
	return true
}

type modelsCommandService struct {
	models *application.ModelService
}

func newModelsCommandService(home string) (AppService, error) {
	models, err := application.NewModelService(application.ModelServiceConfig{Home: home})
	if err != nil {
		return nil, err
	}
	return &modelsCommandService{models: models}, nil
}

func (s *modelsCommandService) StartTranscription(domain.TranscriptionRequest) (string, error) {
	return "", &domain.UserError{Message: "このコマンドでは文字起こしを実行できません", Detail: "models command"}
}

func (s *modelsCommandService) GetJob(string) (domain.TranscriptionJob, error) {
	return domain.TranscriptionJob{}, &domain.UserError{Message: "このコマンドでは文字起こしを実行できません", Detail: "models command"}
}

func (s *modelsCommandService) CancelJob(string) error {
	return &domain.UserError{Message: "このコマンドでは文字起こしを実行できません", Detail: "models command"}
}

func (s *modelsCommandService) ListModels() ([]domain.ModelInfo, error) {
	return s.models.List()
}

func (s *modelsCommandService) InstallModel(ctx context.Context, modelID string) error {
	return s.models.Install(ctx, modelID)
}

func (s *modelsCommandService) GetSettings() (domain.AppSettings, error) {
	return domain.AppSettings{}, &domain.UserError{Message: "このコマンドでは文字起こしを実行できません", Detail: "models command"}
}

func (s *modelsCommandService) Close() {}

func extractInputPath(args []string) (input string, flagArgs []string) {
	for _, arg := range args {
		if input == "" && !strings.HasPrefix(arg, "-") {
			input = arg
			continue
		}
		flagArgs = append(flagArgs, arg)
	}
	return input, flagArgs
}

func parseFormats(value string) ([]domain.ExportFormat, error) {
	parts := strings.Split(value, ",")
	out := make([]domain.ExportFormat, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		format := domain.ExportFormat(part)
		if !format.Valid() {
			return nil, &domain.UserError{Message: "対応していない保存形式です", Detail: part}
		}
		out = append(out, format)
	}
	return out, domain.ValidateFormats(out)
}

func outputFileNames(dir string, formats []domain.ExportFormat) []string {
	names := []string{"transcript.json"}
	for _, format := range formats {
		switch format {
		case domain.FormatTXT:
			names = append(names, "transcript.txt")
		case domain.FormatSRT:
			names = append(names, "transcript.srt")
		case domain.FormatJSON:
			// transcript.json already included
		}
	}
	return names
}

func isUsageError(err error) bool {
	var userErr *domain.UserError
	if !errorsAsUser(err, &userErr) {
		return false
	}
	switch userErr.Message {
	case "対応していないモデルです", "対応していない言語です", "対応していない保存形式です",
		"入力ファイルを指定してください", "保存先を指定してください", "モデルは既にインストールされています":
		return true
	default:
		return false
	}
}

func errorsAsUser(err error, target **domain.UserError) bool {
	var userErr *domain.UserError
	if errors.As(err, &userErr) {
		*target = userErr
		return true
	}
	return false
}
