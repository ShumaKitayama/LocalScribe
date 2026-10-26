package domain

import "slices"

const (
	// ModelLargeV3 は高精度モードのモデル ID である。
	ModelLargeV3 = "large-v3"
	// ModelLargeV3Turbo は高速モードのモデル ID である。
	ModelLargeV3Turbo = "large-v3-turbo"

	modelLargeV3MinBytes      int64 = 2_800_000_000
	modelLargeV3TurboMinBytes int64 = 1_400_000_000
)

// ModelStatus はモデルファイルを推論に使ってよいかを表す。
type ModelStatus string

const (
	ModelMissing   ModelStatus = "missing"
	ModelPartial   ModelStatus = "partial"
	ModelAvailable ModelStatus = "available"
)

// ModelInfo はインストール済みかどうかを含むモデルの説明である。
type ModelInfo struct {
	ID        string      `json:"id"`
	FileName  string      `json:"file_name"`
	SizeBytes int64       `json:"size_bytes,omitzero"`
	Path      string      `json:"path,omitempty"`
	SHA256    string      `json:"sha256,omitempty"`
	Status    ModelStatus `json:"status"`
}

// ValidModelID は MVP の 2 モデルなら true を返す。
func ValidModelID(id string) bool {
	return slices.Contains([]string{ModelLargeV3, ModelLargeV3Turbo}, id)
}

// ModelFileName はモデル ID に対応するファイル名を返す。
func ModelFileName(id string) (string, bool) {
	switch id {
	case ModelLargeV3:
		return "ggml-large-v3.bin", true
	case ModelLargeV3Turbo:
		return "ggml-large-v3-turbo.bin", true
	default:
		return "", false
	}
}

// ModelMinBytes は、これ未満のファイルを未完成として扱う下限を返す。
func ModelMinBytes(id string) (int64, bool) {
	switch id {
	case ModelLargeV3:
		return modelLargeV3MinBytes, true
	case ModelLargeV3Turbo:
		return modelLargeV3TurboMinBytes, true
	default:
		return 0, false
	}
}
