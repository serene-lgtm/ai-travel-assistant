package http_dto

import (
	"fmt"
	"strings"
)

// InspirationSessionCreateRequest captures the REST payload for creating a session.
type InspirationSessionCreateRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (r InspirationSessionCreateRequest) Validate() error {
	if strings.TrimSpace(r.UserID) == "" {
		return fmt.Errorf("user_id is required")
	}
	return nil
}
