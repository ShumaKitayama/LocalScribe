package export

import (
	"encoding/json"
	"fmt"

	"github.com/ShumaKitayama/localscribe/internal/domain"
)

// EncodeTranscriptJSON は設計書 7 節の Transcript JSON を返す。
func EncodeTranscriptJSON(transcript domain.Transcript) ([]byte, error) {
	if err := transcript.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode transcript json: %w", err)
	}
	return append(data, '\n'), nil
}

// DecodeTranscriptJSON は JSON 正本を読み、検証して返す。TXT と SRT は常にこの結果から生成する。
func DecodeTranscriptJSON(data []byte) (domain.Transcript, error) {
	var transcript domain.Transcript
	if err := json.Unmarshal(data, &transcript); err != nil {
		return domain.Transcript{}, &domain.UserError{
			Message: "文字起こし結果を読めませんでした",
			Detail:  err.Error(),
		}
	}
	if err := transcript.Validate(); err != nil {
		return domain.Transcript{}, err
	}
	return transcript, nil
}
