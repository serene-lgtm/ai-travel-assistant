package http_dto

import (
	"fmt"
	"strings"
	"time"

	"ai-reading-assistant/internal/model"
)

// InspirationMessageRequest captures the REST payload from the frontend.
type InspirationMessageRequest struct {
	SessionID           string `json:"session_id" binding:"required"`
	Role                string `json:"role" binding:"required"`
	Kind                string `json:"kind"`
	Content             string `json:"content" binding:"required"`
	SelectedOption      string `json:"selected_option"`
	StartNewInspiration bool   `json:"start_new_inspiration"`
}

// ToModel converts the request into a domain model ready for persistence.
func (r InspirationMessageRequest) ToModel() (*model.InspirationMessage, error) {
	role := model.InspirationMessageRole(r.Role)
	if role != model.InspirationMessageRoleUser && role != model.InspirationMessageRoleAssistant {
		return nil, fmt.Errorf("invalid role %q", r.Role)
	}

	if strings.TrimSpace(r.SessionID) == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if strings.TrimSpace(r.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}

	msg := &model.InspirationMessage{
		SessionID:           r.SessionID,
		Role:                model.InspirationMessageRoleUser,
		Kind:                model.InspirationMessageKind(r.Kind),
		Content:             r.Content,
		StartNewInspiration: r.StartNewInspiration,
		CreatedAt:           time.Now().UTC(),
	}
	if opt := strings.TrimSpace(r.SelectedOption); opt != "" {
		msg.Options = []model.Option{
			{Content: opt, Selected: true},
		}
	}
	return msg, nil
}
