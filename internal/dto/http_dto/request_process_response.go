package http_dto

import (
	"time"

	"ai-reading-assistant/internal/model"
)

// RequestProcessResponse represents the REST response for a request process.
type RequestProcessResponse struct {
	ID        string    `json:"id"`
	Stage     string    `json:"stage"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromModel converts a domain model to a response DTO.
func RequestProcessFromModel(process *model.RequestProcess) *RequestProcessResponse {
	if process == nil {
		return nil
	}

	return &RequestProcessResponse{
		ID:        process.ID,
		Stage:     string(process.Stage),
		CreatedAt: process.CreatedAt,
		UpdatedAt: process.UpdatedAt,
	}
}
