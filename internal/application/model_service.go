package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/audio"
	"github.com/ShumaKitayama/localscribe/internal/domain"
	"github.com/ShumaKitayama/localscribe/internal/platform/macos"
	"github.com/ShumaKitayama/localscribe/internal/storage"
)

const modelInstallHeadroom = 64 * 1024 * 1024

// HTTPDoer はモデル取得の HTTP クライアント差し替え点である。
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type modelManifest struct {
	SchemaVersion int                           `json:"schema_version"`
	Models        map[string]modelManifestEntry `json:"models"`
}

type modelManifestEntry struct {
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// ModelService はモデルの一覧と取得を行う。
type ModelService struct {
	home      string
	catalog   []CatalogEntry
	http      HTTPDoer
	freeBytes func(string) (int64, error)
}

// ModelServiceConfig は ModelService の依存である。
type ModelServiceConfig struct {
	Home      string
	Catalog   []CatalogEntry
	HTTP      HTTPDoer
	FreeBytes func(string) (int64, error)
}

// NewModelService はカタログと HTTP クライアントを束ねる。
func NewModelService(cfg ModelServiceConfig) (*ModelService, error) {
	if strings.TrimSpace(cfg.Home) == "" {
		return nil, &domain.UserError{Message: "ホームディレクトリがありません", Detail: "empty home"}
	}
	catalog := cfg.Catalog
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	httpClient := cfg.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	freeBytes := cfg.FreeBytes
	if freeBytes == nil {
		freeBytes = audio.AvailableBytes
	}
	return &ModelService{
		home:      cfg.Home,
		catalog:   catalog,
		http:      httpClient,
		freeBytes: freeBytes,
	}, nil
}

// List はカタログとローカル状態を合わせて返す。
func (m *ModelService) List() ([]domain.ModelInfo, error) {
	modelDir := macos.ModelsDir(m.home)
	out := make([]domain.ModelInfo, 0, len(m.catalog))
	manifest, _ := loadManifest(modelDir)
	for _, entry := range m.catalog {
		info := domain.ModelInfo{
			ID:       entry.ID,
			FileName: entry.FileName,
			Status:   domain.ModelMissing,
		}
		finalPath := filepath.Join(modelDir, entry.FileName)
		partialPath := finalPath + ".partial"
		if _, err := os.Stat(partialPath); err == nil {
			info.Status = domain.ModelPartial
			if stat, statErr := os.Stat(partialPath); statErr == nil {
				info.SizeBytes = stat.Size()
			}
			out = append(out, info)
			continue
		}
		stat, err := os.Stat(finalPath)
		if err != nil {
			out = append(out, info)
			continue
		}
		info.SizeBytes = stat.Size()
		info.Path = finalPath
		if manifestEntry, ok := manifest.Models[entry.ID]; ok {
			info.SHA256 = manifestEntry.SHA256
		}
		if stat.Size() < entry.MinBytes {
			info.Status = domain.ModelPartial
		} else if err := validateModelFile(entry, finalPath, manifest); err != nil {
			info.Status = domain.ModelPartial
		} else {
			info.Status = domain.ModelAvailable
		}
		out = append(out, info)
	}
	return out, nil
}

// Install は URL から partial へ取得し、検証後に rename する。
func (m *ModelService) Install(ctx context.Context, modelID string) error {
	entry, ok := findCatalogEntry(m.catalog, modelID)
	if !ok {
		return &domain.UserError{Message: "対応していないモデルです", Detail: modelID}
	}
	modelDir := macos.ModelsDir(m.home)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return &domain.UserError{Message: "モデルの保存先を作れません", Detail: err.Error()}
	}
	finalPath := filepath.Join(modelDir, entry.FileName)
	if _, err := os.Stat(finalPath); err == nil {
		return &domain.UserError{Message: "モデルは既にインストールされています", Detail: modelID}
	}
	required := entry.MinBytes + modelInstallHeadroom
	free, err := m.freeBytes(modelDir)
	if err != nil {
		return &domain.UserError{Message: "空き容量を確認できません", Detail: err.Error()}
	}
	if free < required {
		return &domain.UserError{Message: "モデル取得用の空き容量が足りません", Detail: fmt.Sprintf("need %d", required)}
	}

	partialPath := finalPath + ".partial"
	_ = os.Remove(partialPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.URL, nil)
	if err != nil {
		return err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		return &domain.UserError{Message: "モデルを取得できません", Detail: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &domain.UserError{
			Message: "モデルを取得できません",
			Detail:  fmt.Sprintf("status %d", resp.StatusCode),
		}
	}

	file, err := os.Create(partialPath)
	if err != nil {
		return &domain.UserError{Message: "モデルの保存に失敗しました", Detail: err.Error()}
	}
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	written, copyErr := io.Copy(writer, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return &domain.UserError{Message: "モデルの取得が途中で切れました", Detail: copyErr.Error()}
	}
	if closeErr != nil {
		_ = os.Remove(partialPath)
		return &domain.UserError{Message: "モデルの保存に失敗しました", Detail: closeErr.Error()}
	}
	if written < entry.MinBytes {
		_ = os.Remove(partialPath)
		return &domain.UserError{Message: "モデルファイルが不完全です", Detail: fmt.Sprintf("size %d", written)}
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	manifest, err := loadManifest(modelDir)
	if err != nil {
		_ = os.Remove(partialPath)
		return err
	}
	if err := verifyModelSHA256(entry, manifest, sum); err != nil {
		_ = os.Remove(partialPath)
		return err
	}
	manifest.Models[entry.ID] = modelManifestEntry{
		SHA256:    sum,
		SizeBytes: written,
	}
	if err := saveManifest(modelDir, manifest); err != nil {
		_ = os.Remove(partialPath)
		return err
	}
	if err := os.Rename(partialPath, finalPath); err != nil {
		_ = os.Remove(partialPath)
		return &domain.UserError{Message: "モデルの保存に失敗しました", Detail: err.Error()}
	}
	return nil
}

// ValidateModelAvailable は推論に使えるモデルが揃っているかを返す。
func ValidateModelAvailable(home, modelID string) error {
	entry, ok := findCatalogEntry(DefaultCatalog(), modelID)
	if !ok {
		return &domain.UserError{Message: "対応していないモデルです", Detail: modelID}
	}
	modelDir := macos.ModelsDir(home)
	finalPath := filepath.Join(modelDir, entry.FileName)
	partialPath := finalPath + ".partial"
	if _, err := os.Stat(partialPath); err == nil {
		return &domain.UserError{Message: "モデルは取得途中です", Detail: modelID}
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &domain.UserError{Message: "モデルが導入されていません", Detail: modelID}
		}
		return &domain.UserError{Message: "モデルを確認できません", Detail: err.Error()}
	}
	if info.Size() < entry.MinBytes {
		return &domain.UserError{Message: "モデルファイルが不完全です", Detail: modelID}
	}
	manifest, err := loadManifest(modelDir)
	if err != nil {
		return err
	}
	return validateModelFile(entry, finalPath, manifest)
}

func validateModelFile(entry CatalogEntry, finalPath string, manifest modelManifest) error {
	if manifestEntry, ok := manifest.Models[entry.ID]; ok && manifestEntry.SHA256 != "" {
		sum, err := fileSHA256(finalPath)
		if err != nil {
			return err
		}
		if sum != manifestEntry.SHA256 {
			return &domain.UserError{Message: "モデルファイルが不正です", Detail: entry.ID}
		}
	}
	return nil
}

func verifyModelSHA256(entry CatalogEntry, manifest modelManifest, downloaded string) error {
	expected := strings.TrimSpace(entry.SHA256)
	if expected == "" {
		if manifestEntry, ok := manifest.Models[entry.ID]; ok {
			expected = manifestEntry.SHA256
		}
	}
	if expected == "" {
		return nil
	}
	if !strings.EqualFold(downloaded, expected) {
		return &domain.UserError{Message: "モデルファイルのハッシュが一致しません", Detail: entry.ID}
	}
	return nil
}

func loadManifest(modelDir string) (modelManifest, error) {
	path := filepath.Join(modelDir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return modelManifest{
				SchemaVersion: domain.SchemaVersion,
				Models:        map[string]modelManifestEntry{},
			}, nil
		}
		return modelManifest{}, &domain.UserError{Message: "モデル情報を読めません", Detail: err.Error()}
	}
	var manifest modelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return modelManifest{}, &domain.UserError{Message: "モデル情報を読めません", Detail: err.Error()}
	}
	if manifest.Models == nil {
		manifest.Models = map[string]modelManifestEntry{}
	}
	if manifest.SchemaVersion == 0 {
		manifest.SchemaVersion = domain.SchemaVersion
	}
	return manifest, nil
}

func saveManifest(modelDir string, manifest modelManifest) error {
	manifest.SchemaVersion = domain.SchemaVersion
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return &domain.UserError{Message: "モデル情報を保存できません", Detail: err.Error()}
	}
	data = append(data, '\n')
	return storage.WriteFileAtomic(filepath.Join(modelDir, "manifest.json"), data)
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", &domain.UserError{Message: "モデルファイルを読めません", Detail: err.Error()}
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", &domain.UserError{Message: "モデルファイルを読めません", Detail: err.Error()}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
