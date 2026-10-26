package whisper

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/engine"
)

var progressPercentRE = regexp.MustCompile(`progress\s*=\s*(\d+)%`)

// ProcessRunner は whisper-cli を起動する。テストでは偽実装を渡す。
type ProcessRunner interface {
	Run(ctx context.Context, binary string, args []string, onStderrLine func(string)) error
}

// ExecProcessRunner は exec.CommandContext で標準エラーを 1 行ずつ渡す。
type ExecProcessRunner struct{}

// Run は binary を起動し、標準エラーの各行を onStderrLine へ渡す。
func (ExecProcessRunner) Run(ctx context.Context, binary string, args []string, onStderrLine func(string)) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe %s: %w", binary, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", binary, err)
	}

	var stderr bytes.Buffer
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		reader := bufio.NewReader(stderrPipe)
		for {
			line, readErr := reader.ReadString('\n')
			if line != "" {
				trimmed := strings.TrimSuffix(line, "\n")
				stderr.WriteString(line)
				if onStderrLine != nil && trimmed != "" {
					onStderrLine(trimmed)
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	waitErr := cmd.Wait()
	<-scanDone

	if waitErr != nil {
		if ctx.Err() != nil {
			return engine.ErrCancelled
		}
		return fmt.Errorf("%s: %w: %s", binary, waitErr, strings.TrimSpace(stderr.String()))
	}
	if ctx.Err() != nil {
		return engine.ErrCancelled
	}
	return nil
}

// Engine は whisper-cli による認識エンジンである。
type Engine struct {
	Binary string
	Runner ProcessRunner
}

var _ engine.Engine = Engine{}

// NewEngine は whisper-cli のパスを解決してエンジンを作る。
func NewEngine(runner ProcessRunner) (Engine, error) {
	binary, err := ResolveBinary()
	if err != nil {
		return Engine{}, err
	}
	if runner == nil {
		runner = ExecProcessRunner{}
	}
	return Engine{Binary: binary, Runner: runner}, nil
}

// Transcribe は設計書 5 節の引数で whisper-cli を実行し、JSON をパースする。
func (e Engine) Transcribe(ctx context.Context, req engine.Request, onProgress func(engine.Progress)) ([]domain.Segment, error) {
	if strings.TrimSpace(req.WAVPath) == "" {
		return nil, &domain.UserError{Message: "音声ファイルを指定してください", Detail: "empty wav"}
	}
	if strings.TrimSpace(req.ModelPath) == "" {
		return nil, &domain.UserError{Message: "モデルを指定してください", Detail: "empty model"}
	}
	if strings.TrimSpace(req.OutputPrefix) == "" {
		return nil, &domain.UserError{Message: "作業用の出力先を指定してください", Detail: "empty output prefix"}
	}
	args := Args(req)
	err := e.Runner.Run(ctx, e.Binary, args, func(line string) {
		if onProgress == nil {
			return
		}
		onProgress(ParseProgressLine(line))
	})
	if err != nil {
		if errors.Is(err, engine.ErrCancelled) {
			return nil, err
		}
		return nil, &domain.UserError{
			Message: "音声の認識に失敗しました",
			Detail:  err.Error(),
		}
	}
	jsonPath := req.OutputPrefix + ".json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, &domain.UserError{
			Message: "認識結果を読めませんでした",
			Detail:  err.Error(),
		}
	}
	return ParseOutputJSON(data)
}

// Args は whisper-cli の引数配列を返す。実行ファイル名は含めない。
func Args(req engine.Request) []string {
	args := []string{
		"-m", req.ModelPath,
		"-f", req.WAVPath,
		"-l", languageFlag(req.Language),
		"-oj",
		"-of", req.OutputPrefix,
		"-pp",
	}
	if strings.TrimSpace(req.InitialPrompt) != "" {
		args = append(args, "--prompt", req.InitialPrompt)
	}
	return args
}

// ParseProgressLine は標準エラー 1 行を進捗へ変換する。パーセントが無い行は Percent を nil にする。
func ParseProgressLine(line string) engine.Progress {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return engine.Progress{Message: ""}
	}
	if match := progressPercentRE.FindStringSubmatch(trimmed); len(match) == 2 {
		value, err := strconv.Atoi(match[1])
		if err == nil {
			percent := float64(value)
			return engine.Progress{Message: trimmed, Percent: &percent}
		}
	}
	return engine.Progress{Message: trimmed}
}

// ResolveBinary は設計書の解決順で whisper-cli を探す。
func ResolveBinary() (string, error) {
	if path := os.Getenv("LOCALSCRIBE_WHISPER_CLI"); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err == nil {
		appPath := filepath.Join(home, "Library", "Application Support", "LocalScribe", "bin", "whisper-cli")
		if fileExecutable(appPath) {
			return appPath, nil
		}
	}
	repoPath, err := repoBinaryPath()
	if err == nil && fileExecutable(repoPath) {
		return repoPath, nil
	}
	path, lookErr := exec.LookPath("whisper-cli")
	if lookErr != nil {
		return "", &domain.UserError{Message: "whisper-cli が見つかりません", Detail: lookErr.Error()}
	}
	return path, nil
}

func repoBinaryPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		candidate := filepath.Join(dir, "build", "whisper", "whisper-cli")
		if fileExecutable(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repo root not found")
		}
		dir = parent
	}
}

func fileExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Mode()&0o111 != 0
}

func languageFlag(language domain.Language) string {
	switch language {
	case domain.LanguageJA:
		return "ja"
	case domain.LanguageEN:
		return "en"
	case domain.LanguageAuto:
		return "auto"
	default:
		return string(language)
	}
}
