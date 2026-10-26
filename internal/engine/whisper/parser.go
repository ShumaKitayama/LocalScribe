package whisper

import (
	"encoding/json"
	"strings"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

type whisperDocument struct {
	Transcription []whisperSegment `json:"transcription"`
}

type whisperSegment struct {
	Offsets struct {
		From int64 `json:"from"`
		To   int64 `json:"to"`
	} `json:"offsets"`
	Text string `json:"text"`
}

// ParseOutputJSON は whisper-cli の -oj 出力を domain.Segment へ変換する。
// 時刻は offsets（ミリ秒）を正とする。設計書 5 節。
func ParseOutputJSON(data []byte) ([]domain.Segment, error) {
	var doc whisperDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, &domain.UserError{
			Message: "認識結果を読めませんでした",
			Detail:  err.Error(),
		}
	}
	segments := make([]domain.Segment, 0, len(doc.Transcription))
	for _, item := range doc.Transcription {
		segment := domain.Segment{
			StartMS: item.Offsets.From,
			EndMS:   item.Offsets.To,
			Text:    strings.TrimSpace(item.Text),
		}
		if err := segment.Validate(); err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}
	return segments, nil
}
