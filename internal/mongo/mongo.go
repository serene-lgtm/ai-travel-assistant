package mongo

import (
	"context"
	"fmt"
	"sync"

	"ai-reading-assistant/internal/config"

	driver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client wraps the official MongoDB client with the configured database name.
type Client struct {
	client *driver.Client
	dbName string
}

var (
	globalClient *Client
	initErr      error
	once         sync.Once
)

// Connect establishes a new MongoDB client based on the provided configuration.
func Connect(ctx context.Context, cfg config.MongoConfig) (*Client, error) {
	ctx = ensureContext(ctx)

	connectCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	mClient, err := driver.Connect(connectCtx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer pingCancel()
	if err := mClient.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = mClient.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return &Client{
		client: mClient,
		dbName: cfg.Database,
	}, nil
}

// Global returns a singleton MongoDB client using configuration loaded at init.
func Global(ctx context.Context) (*Client, error) {
	once.Do(func() {
		globalClient, initErr = Connect(ctx, config.Global().Mongo)
	})
	return globalClient, initErr
}

// Database returns the configured database handle from the client.
func (c *Client) Database() *driver.Database {
	return c.client.Database(c.dbName)
}

// Collection returns a specific collection from the configured database.
func (c *Client) Collection(name string) *driver.Collection {
	return c.Database().Collection(name)
}

// Client exposes the underlying mongo-driver client.
func (c *Client) Client() *driver.Client {
	return c.client
}

// Disconnect closes the MongoDB connection.
func (c *Client) Disconnect(ctx context.Context) error {
	return c.client.Disconnect(ensureContext(ctx))
}

// CloseGlobal disconnects the singleton client, if initialized.
func CloseGlobal(ctx context.Context) error {
	if globalClient == nil {
		return nil
	}
	err := globalClient.Disconnect(ctx)
	globalClient = nil
	once = sync.Once{}
	return err
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
