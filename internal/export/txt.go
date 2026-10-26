package export

import (
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

// RenderTXT は表示文を改行で連結する。edited_text があればそれを、無ければ text を使う。
func RenderTXT(transcript domain.Transcript) string {
	if len(transcript.Segments) == 0 {
		return ""
	}
	lines := make([]string, len(transcript.Segments))
	for i, segment := range transcript.Segments {
		lines[i] = segment.DisplayText()
	}
	return strings.Join(lines, "\n")
}

// TXTFromJSON は JSON 正本を読み直してから TXT を生成する。
func TXTFromJSON(data []byte) (string, error) {
	transcript, err := DecodeTranscriptJSON(data)
	if err != nil {
		return "", err
	}
	return RenderTXT(transcript), nil
}
