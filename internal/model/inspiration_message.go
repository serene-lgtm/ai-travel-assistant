package model

import "time"

type InspirationMessageRole string

const (
	InspirationMessageRoleUser      InspirationMessageRole = "user"
	InspirationMessageRoleAssistant InspirationMessageRole = "assistant"
)

type InspirationMessageKind string

const (
	MessageKindUserInput  InspirationMessageKind = "user_input"
	MessageKindUserChoice InspirationMessageKind = "user_choice"
	MessageKindClarifyAsk InspirationMessageKind = "clarify_question"
	MessageKindAssistant  InspirationMessageKind = "assistant_reply"
)

type InspirationMessage struct {
	ID                  string                 `json:"id" bson:"_id,omitempty"`
	SessionID           string                 `json:"session_id" bson:"sid"` // TravelSession ID
	Role                InspirationMessageRole `json:"role" bson:"role"`
	Kind                InspirationMessageKind `json:"kind,omitempty" bson:"kind,omitempty"`
	Content             string                 `json:"content" bson:"cnt"`
	Options             []Option               `json:"options,omitempty" bson:"opts,omitempty"`
	StartNewInspiration bool                   `json:"start_new_inspiration,omitempty" bson:"-"`
	CreatedAt           time.Time              `json:"created_at" bson:"cat"`
}

type Option struct {
	Content  string `json:"content" bson:"cnt"`
	Selected bool   `json:"selected" bson:"sel"`
}
