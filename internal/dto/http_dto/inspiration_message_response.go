package http_dto

// InspirationMessageResponse returns the REST payload to the frontend.
type InspirationMessageResponse struct {
	SessionID      string           `json:"session_id" binding:"required"`
	Role           string           `json:"role" binding:"required"`
	Kind           string           `json:"kind"` // user_input, user_choice, clarify_question, assistant_reply
	Content        string           `json:"content" binding:"required"`
	Options        []ResponseOption `json:"options,omitempty"`
	TargetField    string           `json:"target_field,omitempty"`
	Warning        string           `json:"warning"` // to warn the limitation
}

type ResponseOption struct {
	Content  string `json:"content"`
	Selected bool   `json:"selected"`
}
