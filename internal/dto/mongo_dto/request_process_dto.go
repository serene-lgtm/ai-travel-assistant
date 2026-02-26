package mongo_dto

import (
	"fmt"
	"time"

	"ai-reading-assistant/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RequestProcessDTO maps the RequestProcess model to Mongo format.
type RequestProcessDTO struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Stage     string             `bson:"stg"`
	CreatedAt time.Time          `bson:"cat"`
	UpdatedAt time.Time          `bson:"uat"`
}

// RequestProcessToDTO converts a model.RequestProcess to DTO form.
func RequestProcessToDTO(process *model.RequestProcess) (*RequestProcessDTO, error) {
	if process == nil {
		return nil, fmt.Errorf("process is nil")
	}

	processID, err := objectIDFromHex(process.ID)
	if err != nil {
		return nil, fmt.Errorf("process id: %w", err)
	}

	return &RequestProcessDTO{
		ID:        processID,
		Stage:     string(process.Stage),
		CreatedAt: process.CreatedAt,
		UpdatedAt: process.UpdatedAt,
	}, nil
}

// RequestProcessFromDTO converts RequestProcessDTO back to the domain model.
func RequestProcessFromDTO(dto *RequestProcessDTO) (*model.RequestProcess, error) {
	if dto == nil {
		return nil, fmt.Errorf("process dto is nil")
	}

	return &model.RequestProcess{
		ID:        hexFromObjectID(dto.ID),
		Stage:     model.RequestStage(dto.Stage),
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}, nil
}
