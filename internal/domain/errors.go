package domain

// UserError は利用者に見せる文と、ログに残す詳細を分ける。
type UserError struct {
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Error は利用者向けの文を返す。
func (e *UserError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
