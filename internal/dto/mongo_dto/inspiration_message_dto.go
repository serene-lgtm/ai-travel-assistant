package mongo_dto

import (
	"fmt"
	"time"

	"ai-reading-assistant/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InspirationMessageDTO maps InspirationMessage model to MongoDB format.
type InspirationMessageDTO struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	SessionID primitive.ObjectID `bson:"sid"`
	Role      string             `bson:"role"`
	Kind      string             `bson:"kind,omitempty"`
	Options   []mongoOption      `bson:"opts,omitempty"`
	Content   string             `bson:"cnt"`
	CreatedAt time.Time          `bson:"cat"`
}

type mongoOption struct {
	Content  string `bson:"cnt"`
	Selected bool   `bson:"sel"`
}

// InspirationMessageToDTO converts a model.InspirationMessage into DTO form.
func InspirationMessageToDTO(msg *model.InspirationMessage) (*InspirationMessageDTO, error) {
	if msg == nil {
		return nil, fmt.Errorf("conversation message is nil")
	}

	id, err := objectIDFromHex(msg.ID)
	if err != nil {
		return nil, fmt.Errorf("message id: %w", err)
	}
	sessionID, err := objectIDFromHex(msg.SessionID)
	if err != nil {
		return nil, fmt.Errorf("message session id: %w", err)
	}
	return &InspirationMessageDTO{
		ID:        id,
		SessionID: sessionID,
		Role:      string(msg.Role),
		Kind:      string(msg.Kind),
		Options:   encodeOptions(msg.Options),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}, nil
}

// InspirationMessageFromDTO converts the DTO back to the domain model.
func InspirationMessageFromDTO(dto *InspirationMessageDTO) (*model.InspirationMessage, error) {
	if dto == nil {
		return nil, fmt.Errorf("conversation message dto is nil")
	}

	return &model.InspirationMessage{
		ID:        hexFromObjectID(dto.ID),
		SessionID: hexFromObjectID(dto.SessionID),
		Role:      model.InspirationMessageRole(dto.Role),
		Kind:      model.InspirationMessageKind(dto.Kind),
		Options:   decodeOptions(dto.Options),
		Content:   dto.Content,
		CreatedAt: dto.CreatedAt,
	}, nil
}

func encodeOptions(opts []model.Option) []mongoOption {
	if len(opts) == 0 {
		return nil
	}
	out := make([]mongoOption, 0, len(opts))
	for _, o := range opts {
		out = append(out, mongoOption{
			Content:  o.Content,
			Selected: o.Selected,
		})
	}
	return out
}

func decodeOptions(opts []mongoOption) []model.Option {
	if len(opts) == 0 {
		return nil
	}
	out := make([]model.Option, 0, len(opts))
	for _, o := range opts {
		out = append(out, model.Option{
			Content:  o.Content,
			Selected: o.Selected,
		})
	}
	return out
}
