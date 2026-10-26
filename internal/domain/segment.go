package domain

// Segment は 1 区切りの認識結果である。Text は認識直後の文で、編集では変えない。
type Segment struct {
	StartMS    int64   `json:"start_ms"`
	EndMS      int64   `json:"end_ms"`
	Text       string  `json:"text"`
	EditedText *string `json:"edited_text"`
}

// DisplayText は、編集後の文があればそれを、無ければ認識直後の文を返す。
// 空文字への編集は未編集とは別であり、認識文へ戻さない。
func (s Segment) DisplayText() string {
	if s.EditedText != nil {
		return *s.EditedText
	}
	return s.Text
}

// ApplyEdit は編集後の文だけを更新する。
func (s *Segment) ApplyEdit(edited string) {
	s.EditedText = &edited
}

// Validate は時刻が負でなく、終了が開始以上であることを確認する。
func (s Segment) Validate() error {
	if s.StartMS < 0 || s.EndMS < s.StartMS {
		return &UserError{Message: "時刻の範囲が不正です", Detail: "segment time"}
	}
	return nil
}

// SegmentEdit は 1 セグメントの編集指示である。
type SegmentEdit struct {
	Index      int    `json:"index"`
	EditedText string `json:"edited_text"`
}
