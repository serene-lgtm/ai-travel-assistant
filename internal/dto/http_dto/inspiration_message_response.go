package http_dto

// InspirationMessageResponse returns the REST payload to the frontend.
type InspirationMessageResponse struct {
	SessionID     string              `json:"session_id" binding:"required"`
	Role          string              `json:"role" binding:"required"`
	Kind          string              `json:"kind"` // user_input, user_choice, clarify_question, assistant_reply
	Content       string              `json:"content" binding:"required"`
	Options       []ResponseOption    `json:"options,omitempty"`
	TargetField   string              `json:"target_field,omitempty"`
	Warning       string              `json:"warning"` // to warn the limitation
	IsInspiration bool                `json:"is_inspiration"`
	Inspiration   *InspirationPayload `json:"inspiration,omitempty"`
}

type ResponseOption struct {
	Content  string `json:"content"`
	Selected bool   `json:"selected"`
}

type InspirationPayload struct {
	ID         string           `json:"id"`
	Output     string           `json:"output"`
	Keywords   []KeywordPayload `json:"keywords,omitempty"`
	IsFavorite bool             `json:"is_favorite"`
}

type KeywordPayload struct {
	Content        string                 `json:"content"`
	Start          string                 `json:"start"`
	End            string                 `json:"end"`
	WikiDefinition *WikiDefinitionPayload `json:"wiki_definition,omitempty"`
}

type WikiDefinitionPayload struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	FullURL string `json:"full_url"`
}
