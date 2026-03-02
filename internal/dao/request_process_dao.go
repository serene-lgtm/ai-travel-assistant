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
	"go.mongodb.org/mongo-driver/mongo/options"
)

const requestProcessCollection = "request_process"

// RequestProcessDao defines persistence operations for request progress tracking.
type RequestProcessDao interface {
	Create(ctx context.Context, process *model.RequestProcess) (*model.RequestProcess, error)
	Update(ctx context.Context, process *model.RequestProcess) (*model.RequestProcess, error)
	GetByID(ctx context.Context, id string) (*model.RequestProcess, error)
	GetBySessionID(ctx context.Context, sessionID string) (*model.RequestProcess, error)
}

type requestProcessDao struct {
	client *imongo.Client
}

// NewRequestProcessDao constructs a DAO backed by MongoDB.
func NewRequestProcessDao(client *imongo.Client) RequestProcessDao {
	return &requestProcessDao{client: client}
}

// Create inserts a new request process record into MongoDB.
func (d *requestProcessDao) Create(ctx context.Context, process *model.RequestProcess) (*model.RequestProcess, error) {
	if process == nil {
		return nil, fmt.Errorf("request process is nil")
	}
	if process.SessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if process.CreatedAt.IsZero() {
		process.CreatedAt = time.Now().UTC()
	}
	if process.UpdatedAt.IsZero() {
		process.UpdatedAt = process.CreatedAt
	}

	processDTO, err := mongo_dto.RequestProcessToDTO(process)
	if err != nil {
		return nil, fmt.Errorf("process dto: %w", err)
	}
	if processDTO.ID == primitive.NilObjectID {
		processDTO.ID = primitive.NewObjectID()
		process.ID = processDTO.ID.Hex()
	}

	result, err := d.collection().InsertOne(ctx, processDTO)
	if err != nil {
		return nil, fmt.Errorf("insert request process: %w", err)
	}

	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		process.ID = insertedID.Hex()
	}
	return process, nil
}

// Update persists changes to a request process record.
func (d *requestProcessDao) Update(ctx context.Context, process *model.RequestProcess) (*model.RequestProcess, error) {
	if process == nil {
		return nil, fmt.Errorf("request process is nil")
	}
	if process.ID == "" {
		return nil, fmt.Errorf("request process id is empty")
	}

	process.UpdatedAt = time.Now().UTC()

	processDTO, err := mongo_dto.RequestProcessToDTO(process)
	if err != nil {
		return nil, fmt.Errorf("process dto: %w", err)
	}
	if processDTO.ID == primitive.NilObjectID {
		return nil, fmt.Errorf("invalid request process id %q", process.ID)
	}

	filter := bson.M{"_id": processDTO.ID}
	update := bson.M{"$set": bson.M{
		"stg": processDTO.Stage,
		"sat": processDTO.StartedAt,
		"cat": processDTO.CompletedAt,
		"uat": processDTO.UpdatedAt,
		"err": processDTO.Error,
	}}

	result, err := d.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("update request process: %w", err)
	}
	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("request process %s not found", process.ID)
	}
	return process, nil
}

// GetByID retrieves a request process by its ID.
func (d *requestProcessDao) GetByID(ctx context.Context, id string) (*model.RequestProcess, error) {
	if id == "" {
		return nil, fmt.Errorf("id is empty")
	}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id %q: %w", id, err)
	}

	var processDTO mongo_dto.RequestProcessDTO
	if err := d.collection().FindOne(ctx, bson.M{"_id": objID}).Decode(&processDTO); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("request process %s not found", id)
		}
		return nil, fmt.Errorf("find request process: %w", err)
	}

	modelProcess, err := mongo_dto.RequestProcessFromDTO(&processDTO)
	if err != nil {
		return nil, fmt.Errorf("process from dto: %w", err)
	}
	return modelProcess, nil
}

// GetBySessionID retrieves the most recent request process for a session.
func (d *requestProcessDao) GetBySessionID(ctx context.Context, sessionID string) (*model.RequestProcess, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session id is empty")
	}

	var processDTO mongo_dto.RequestProcessDTO
	opts := options.FindOne().SetSort(bson.M{"cat_at": -1})

	if err := d.collection().FindOne(ctx, bson.M{"sid": sessionID}, opts).Decode(&processDTO); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("request process for session %s not found", sessionID)
		}
		return nil, fmt.Errorf("find request process by session: %w", err)
	}

	modelProcess, err := mongo_dto.RequestProcessFromDTO(&processDTO)
	if err != nil {
		return nil, fmt.Errorf("process from dto: %w", err)
	}
	return modelProcess, nil
}

func (d *requestProcessDao) collection() *mongo.Collection {
	return d.client.Collection(requestProcessCollection)
}
