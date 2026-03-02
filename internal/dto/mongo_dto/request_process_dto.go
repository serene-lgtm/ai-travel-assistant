package mongo_dto

import (
	"fmt"
	"time"

	"ai-reading-assistant/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RequestProcessDTO maps the RequestProcess model to Mongo format.
type RequestProcessDTO struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	SessionID   string             `bson:"sid"`
	UserID      string             `bson:"uid"`
	Stage       string             `bson:"stg"`
	StartedAt   time.Time          `bson:"sat"`
	CompletedAt time.Time          `bson:"cat"`
	CreatedAt   time.Time          `bson:"cat_at"`
	UpdatedAt   time.Time          `bson:"uat"`
	Error       string             `bson:"err,omitempty"`
}

// RequestProcessToDTO converts a model.RequestProcess to DTO form.
func RequestProcessToDTO(process *model.RequestProcess) (*RequestProcessDTO, error) {
	if process == nil {
		return nil, fmt.Errorf("process is nil")
	}

	processID, err := objectIDFromHex(process.ID)
	if err != nil && process.ID != "" {
		return nil, fmt.Errorf("process id: %w", err)
	}

	return &RequestProcessDTO{
		ID:          processID,
		SessionID:   process.SessionID,
		UserID:      process.UserID,
		Stage:       string(process.Stage),
		StartedAt:   process.StartedAt,
		CompletedAt: process.CompletedAt,
		CreatedAt:   process.CreatedAt,
		UpdatedAt:   process.UpdatedAt,
		Error:       process.Error,
	}, nil
}

// RequestProcessFromDTO converts RequestProcessDTO back to the domain model.
func RequestProcessFromDTO(dto *RequestProcessDTO) (*model.RequestProcess, error) {
	if dto == nil {
		return nil, fmt.Errorf("process dto is nil")
	}

	return &model.RequestProcess{
		ID:          hexFromObjectID(dto.ID),
		SessionID:   dto.SessionID,
		UserID:      dto.UserID,
		Stage:       model.RequestStage(dto.Stage),
		StartedAt:   dto.StartedAt,
		CompletedAt: dto.CompletedAt,
		CreatedAt:   dto.CreatedAt,
		UpdatedAt:   dto.UpdatedAt,
		Error:       dto.Error,
	}, nil
}
