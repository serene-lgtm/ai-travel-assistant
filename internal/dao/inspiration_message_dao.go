package dao

import (
	"context"
	"fmt"
	"time"

	mongo_dto "ai-reading-assistant/internal/dto/mongo_dto"
	imongo "ai-reading-assistant/internal/mongo"
	"ai-reading-assistant/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const inspirationMessageCollection = "inspiration_message"

// InspirationMessageDao persists InspirationMessage documents.
type InspirationMessageDao interface {
	Create(ctx context.Context, message *model.InspirationMessage) (*model.InspirationMessage, error)
	GetByIDs(ctx context.Context, ids []string) ([]*model.InspirationMessage, error)
}

type inspirationMessageDao struct {
	client *imongo.Client
}

// NewInspirationMessageDao constructs a DAO backed by Mongo.
func NewInspirationMessageDao(client *imongo.Client) InspirationMessageDao {
	return &inspirationMessageDao{client: client}
}

// Create writes an inspiration message document.
func (d *inspirationMessageDao) Create(ctx context.Context, message *model.InspirationMessage) (*model.InspirationMessage, error) {
	if message == nil {
		return nil, fmt.Errorf("inspiration message is nil")
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	messageDTO, err := mongo_dto.InspirationMessageToDTO(message)
	if err != nil {
		return nil, fmt.Errorf("message dto: %w", err)
	}
	if messageDTO.ID == primitive.NilObjectID {
		messageDTO.ID = primitive.NewObjectID()
		message.ID = messageDTO.ID.Hex()
	}

	result, err := d.collection().InsertOne(ctx, messageDTO)
	if err != nil {
		return nil, fmt.Errorf("insert inspiration message: %w", err)
	}

	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		message.ID = insertedID.Hex()
	}
	return message, nil
}

// GetByIDs returns messages in the same order as the provided IDs slice.
func (d *inspirationMessageDao) GetByIDs(ctx context.Context, ids []string) ([]*model.InspirationMessage, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, fmt.Errorf("invalid inspiration message id %q: %w", id, err)
		}
		objectIDs = append(objectIDs, objID)
	}

	cursor, err := d.collection().Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, fmt.Errorf("find inspiration messages: %w", err)
	}
	defer cursor.Close(ctx)

	messageMap := make(map[string]*model.InspirationMessage, len(ids))
	for cursor.Next(ctx) {
		var dto mongo_dto.InspirationMessageDTO
		if err := cursor.Decode(&dto); err != nil {
			return nil, fmt.Errorf("decode inspiration message: %w", err)
		}
		msg, err := mongo_dto.InspirationMessageFromDTO(&dto)
		if err != nil {
			return nil, fmt.Errorf("message from dto: %w", err)
		}
		messageMap[msg.ID] = msg
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate inspiration messages: %w", err)
	}

	ordered := make([]*model.InspirationMessage, 0, len(ids))
	for _, id := range ids {
		if msg, ok := messageMap[id]; ok {
			ordered = append(ordered, msg)
		}
	}
	return ordered, nil
}

func (d *inspirationMessageDao) collection() *mongo.Collection {
	return d.client.Collection(inspirationMessageCollection)
}
