package mongo_dto

import (
	"fmt"
	"time"

	"ai-reading-assistant/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InspirationSessionDTO maps the InspirationSession model to Mongo format.
type InspirationSessionDTO struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	UserID       primitive.ObjectID `bson:"uid,omitempty"`
	MaxToken     int                `bson:"mt"`
	Messages     []string           `bson:"msgs"`
	Inspirations []mongoInspiration `bson:"rqmt,omitempty"`
	Status       string             `bson:"st"`
	CreatedAt    time.Time          `bson:"cat"`
}

type mongoRequirementItem struct {
	Content        string `bson:"cnt"`
	Score          int    `bson:"score"`
	SelectedOption string `bson:"sel,omitempty"`
}

type mongoInspiration struct {
	ID         string               `bson:"id"`
	UserID     primitive.ObjectID   `bson:"uid,omitempty"`
	Mood       mongoRequirementItem `bson:"mood"`
	Scene      mongoRequirementItem `bson:"scene"`
	Focus      mongoRequirementItem `bson:"focus"`
	Output     string               `bson:"op,omitempty"`
	IsFavorite bool                 `bson:"if"`
}

// InspirationSessionToDTO converts a model.InspirationSession to DTO form.
func InspirationSessionToDTO(session *model.InspirationSession) (*InspirationSessionDTO, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	sessionID, err := objectIDFromHex(session.ID)
	if err != nil {
		return nil, fmt.Errorf("session id: %w", err)
	}
	userID, err := objectIDFromHex(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("session user id: %w", err)
	}

	return &InspirationSessionDTO{
		ID:           sessionID,
		UserID:       userID,
		MaxToken:     session.MaxToken,
		Messages:     append([]string(nil), session.Messages...),
		Status:       string(session.Status),
		Inspirations: encodeInspirations(session.Inspirations),
		CreatedAt:    session.CreatedAt,
	}, nil
}

// InspirationSessionFromDTO converts InspirationSessionDTO back to the domain model.
func InspirationSessionFromDTO(dto *InspirationSessionDTO) (*model.InspirationSession, error) {
	if dto == nil {
		return nil, fmt.Errorf("session dto is nil")
	}

	return &model.InspirationSession{
		ID:           hexFromObjectID(dto.ID),
		UserID:       hexFromObjectID(dto.UserID),
		MaxToken:     dto.MaxToken,
		Messages:     append([]string(nil), dto.Messages...),
		Status:       model.SessionStatus(dto.Status),
		Inspirations: decodeInspirations(dto.Inspirations),
		CreatedAt:    dto.CreatedAt,
	}, nil
}

func encodeInspirations(items []model.Inspiration) []mongoInspiration {
	if len(items) == 0 {
		return nil
	}
	out := make([]mongoInspiration, 0, len(items))
	for _, v := range items {
		userID, _ := objectIDFromHex(v.UserID)
		out = append(out, mongoInspiration{
			ID:         v.ID,
			UserID:     userID,
			Mood:       toMongoRequirementItem(v.Mood),
			Scene:      toMongoRequirementItem(v.Scene),
			Focus:      toMongoRequirementItem(v.Focus),
			Output:     v.Output,
			IsFavorite: v.IsFavorite,
		})
	}
	return out
}

func decodeInspirations(raw []mongoInspiration) []model.Inspiration {
	if len(raw) == 0 {
		return make([]model.Inspiration, 0)
	}
	out := make([]model.Inspiration, 0, len(raw))
	for _, v := range raw {
		out = append(out, model.Inspiration{
			ID:         v.ID,
			UserID:     hexFromObjectID(v.UserID),
			Mood:       fromMongoRequirementItem(v.Mood),
			Scene:      fromMongoRequirementItem(v.Scene),
			Focus:      fromMongoRequirementItem(v.Focus),
			Output:     v.Output,
			IsFavorite: v.IsFavorite,
		})
	}
	return out
}

func toMongoRequirementItem(v model.RequirementItem) mongoRequirementItem {
	return mongoRequirementItem{
		Content:        v.Content,
		Score:          v.Score,
		SelectedOption: v.SelectedOption,
	}
}

func fromMongoRequirementItem(v mongoRequirementItem) model.RequirementItem {
	return model.RequirementItem{
		Content:        v.Content,
		Score:          v.Score,
		SelectedOption: v.SelectedOption,
	}
}

func objectIDFromHex(id string) (primitive.ObjectID, error) {
	if id == "" {
		return primitive.NilObjectID, nil
	}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid object id %q: %w", id, err)
	}
	return objID, nil
}

func hexFromObjectID(id primitive.ObjectID) string {
	if id == primitive.NilObjectID {
		return ""
	}
	return id.Hex()
}
