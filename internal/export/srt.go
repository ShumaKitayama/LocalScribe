package export

import (
	"fmt"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

// RenderSRT は表示文とミリ秒時刻から SRT を生成する。通し番号は 1 始まり。
// 各字幕のあとには空行を置く。これが字幕の区切りになる。
func RenderSRT(transcript domain.Transcript) string {
	if len(transcript.Segments) == 0 {
		return ""
	}
	var b strings.Builder
	for i, segment := range transcript.Segments {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n",
			i+1,
			formatSRTTimestamp(segment.StartMS),
			formatSRTTimestamp(segment.EndMS),
			segment.DisplayText(),
		)
	}
	return b.String()
}

// SRTFromJSON は JSON 正本を読み直してから SRT を生成する。
func SRTFromJSON(data []byte) (string, error) {
	transcript, err := DecodeTranscriptJSON(data)
	if err != nil {
		return "", err
	}
	return RenderSRT(transcript), nil
}

func formatSRTTimestamp(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	hours := ms / 3_600_000
	ms %= 3_600_000
	minutes := ms / 60_000
	ms %= 60_000
	seconds := ms / 1_000
	millis := ms % 1_000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}
