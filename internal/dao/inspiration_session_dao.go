package dao

import (
	"context"
	"fmt"
	"time"

	mongo_dto "ai-reading-assistant/internal/dto/mongo_dto"
	"ai-reading-assistant/internal/model"
	imongo "ai-reading-assistant/internal/mongo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const inspirationSessionCollection = "inspiration_session"

// InspirationSessionDao defines persistence operations for inspiration sessions.
type InspirationSessionDao interface {
	Create(ctx context.Context, session *model.InspirationSession) (*model.InspirationSession, error)
	Update(ctx context.Context, session *model.InspirationSession) (*model.InspirationSession, error)
	GetByID(ctx context.Context, id string) (*model.InspirationSession, error)
}

type inspirationSessionDao struct {
	client *imongo.Client
}

// NewInspirationSessionDao constructs a DAO backed by MongoDB.
func NewInspirationSessionDao(client *imongo.Client) InspirationSessionDao {
	return &inspirationSessionDao{client: client}
}

// Create inserts the provided inspiration session into MongoDB.
func (d *inspirationSessionDao) Create(ctx context.Context, session *model.InspirationSession) (*model.InspirationSession, error) {
	if session == nil {
		return nil, fmt.Errorf("inspiration session is nil")
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	sessionDTO, err := mongo_dto.InspirationSessionToDTO(session)
	if err != nil {
		return nil, fmt.Errorf("session dto: %w", err)
	}
	if sessionDTO.ID == primitive.NilObjectID {
		sessionDTO.ID = primitive.NewObjectID()
		session.ID = sessionDTO.ID.Hex()
	}

	result, err := d.collection().InsertOne(ctx, sessionDTO)
	if err != nil {
		return nil, fmt.Errorf("insert inspiration session: %w", err)
	}

	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		session.ID = insertedID.Hex()
	}
	return session, nil
}

// Update persists changes on the provided inspiration session.
func (d *inspirationSessionDao) Update(ctx context.Context, session *model.InspirationSession) (*model.InspirationSession, error) {
	if session == nil {
		return nil, fmt.Errorf("inspiration session is nil")
	}
	if session.ID == "" {
		return nil, fmt.Errorf("inspiration session id is empty")
	}

	sessionDTO, err := mongo_dto.InspirationSessionToDTO(session)
	if err != nil {
		return nil, fmt.Errorf("session dto: %w", err)
	}
	if sessionDTO.ID == primitive.NilObjectID {
		return nil, fmt.Errorf("invalid inspiration session id %q", session.ID)
	}

	filter := bson.M{"_id": sessionDTO.ID}
	update := bson.M{"$set": bson.M{
		"uid":  sessionDTO.UserID,
		"msgs": sessionDTO.Messages,
		"rqmt": sessionDTO.Inspirations,
		"st":   sessionDTO.Status,
	}}

	result, err := d.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("update inspiration session: %w", err)
	}
	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("inspiration session %s not found", session.ID)
	}
	return session, nil
}

func (d *inspirationSessionDao) collection() *mongo.Collection {
	return d.client.Collection(inspirationSessionCollection)
}

// GetByID loads an inspiration session document by its hex string ID.
func (d *inspirationSessionDao) GetByID(ctx context.Context, id string) (*model.InspirationSession, error) {
	if id == "" {
		return nil, fmt.Errorf("id is empty")
	}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q: %w", id, err)
	}

	var sessionDTO mongo_dto.InspirationSessionDTO
	if err := d.collection().FindOne(ctx, bson.M{"_id": objID}).Decode(&sessionDTO); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("inspiration session %s not found", id)
		}
		return nil, fmt.Errorf("find inspiration session: %w", err)
	}

	modelSession, err := mongo_dto.InspirationSessionFromDTO(&sessionDTO)
	if err != nil {
		return nil, fmt.Errorf("session from dto: %w", err)
	}
	return modelSession, nil
}
