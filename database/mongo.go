package database

import (
	"context"
	"time"

	"aviator-backend/config"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client *mongo.Client
	DB     *mongo.Database
)

// Connect establishes MongoDB connection with connection pooling
func Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(config.AppConfig.MongoURI).
		SetMaxPoolSize(config.AppConfig.MongoMaxPool).
		SetMinPoolSize(config.AppConfig.MongoMinPool).
		SetMaxConnIdleTime(30 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	Client = client
	DB = client.Database(config.AppConfig.MongoDB)

	log.Info().
		Str("host", config.AppConfig.MongoURI).
		Uint64("max_pool", config.AppConfig.MongoMaxPool).
		Uint64("min_pool", config.AppConfig.MongoMinPool).
		Msg("db_connected")

	// Create indexes
	return createIndexes(ctx)
}

// Disconnect closes the MongoDB connection
func Disconnect() error {
	if Client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return Client.Disconnect(ctx)
}
