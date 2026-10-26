package application

import (
	"github.com/ShumaKitayama/localscribe/internal/domain"
)

// CatalogEntry は配布カタログの 1 モデルである。SHA256 が空なら初回取得時に manifest へ記録する。
type CatalogEntry struct {
	ID       string
	FileName string
	URL      string
	MinBytes int64
	SHA256   string
}

const (
	catalogLargeV3URL      = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin"
	catalogLargeV3TurboURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin"
)

// DefaultCatalog は設計書 8 節の 2 モデルを返す。
func DefaultCatalog() []CatalogEntry {
	largeMin, _ := domain.ModelMinBytes(domain.ModelLargeV3)
	turboMin, _ := domain.ModelMinBytes(domain.ModelLargeV3Turbo)
	largeFile, _ := domain.ModelFileName(domain.ModelLargeV3)
	turboFile, _ := domain.ModelFileName(domain.ModelLargeV3Turbo)
	return []CatalogEntry{
		{
			ID:       domain.ModelLargeV3,
			FileName: largeFile,
			URL:      catalogLargeV3URL,
			MinBytes: largeMin,
		},
		{
			ID:       domain.ModelLargeV3Turbo,
			FileName: turboFile,
			URL:      catalogLargeV3TurboURL,
			MinBytes: turboMin,
		},
	}
}

func findCatalogEntry(catalog []CatalogEntry, modelID string) (CatalogEntry, bool) {
	for _, entry := range catalog {
		if entry.ID == modelID {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}
