package mongo_dto

import (
	"fmt"

	"ai-reading-assistant/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserDTO maps the User model to Mongo format.
type UserDTO struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Username string             `bson:"un"`
	Password string             `bson:"pwd,omitempty"`
	Email    string             `bson:"email"`
}

// UserToDTO converts a model.User to DTO form.
func UserToDTO(user *model.User) (*UserDTO, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}

	userID, err := objectIDFromHex(user.ID)
	if err != nil {
		return nil, fmt.Errorf("user id: %w", err)
	}

	return &UserDTO{
		ID:       userID,
		Username: user.Username,
		Password: user.Password,
		Email:    user.Email,
	}, nil
}

// UserFromDTO converts UserDTO back to the domain model.
func UserFromDTO(dto *UserDTO) (*model.User, error) {
	if dto == nil {
		return nil, fmt.Errorf("user dto is nil")
	}

	return &model.User{
		ID:       hexFromObjectID(dto.ID),
		Username: dto.Username,
		Password: dto.Password,
		Email:    dto.Email,
	}, nil
}
