package http_dto

import (
	"fmt"
	"strings"
)

type InspirationFavoriteRequest struct {
	SessionID     string `json:"session_id"`
	InspirationID string `json:"inspiration_id"`
}

func (r InspirationFavoriteRequest) Validate() error {
	if strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("session_id is required")
	}
	if strings.TrimSpace(r.InspirationID) == "" {
		return fmt.Errorf("inspiration_id is required")
	}
	return nil
}
