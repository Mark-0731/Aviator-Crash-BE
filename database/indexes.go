package database

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// indexDefinition represents a collection and its indexes
type indexDefinition struct {
	collection string
	indexes    []mongo.IndexModel
}

// getAllIndexes returns all index definitions for the database
func getAllIndexes() []indexDefinition {
	return []indexDefinition{
		{
			collection: "users",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "registration_ip", Value: 1}}},
			},
		},
		{
			collection: "bets",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "user_id", Value: 1}}},
				{Keys: bson.D{{Key: "round_id", Value: 1}}},
				{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "round_id", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "status", Value: 1}}},
			},
		},
		{
			collection: "rounds",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "round_id", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "status", Value: 1}}},
				{Keys: bson.D{{Key: "started_at", Value: -1}}},
			},
		},
		{
			collection: "transactions",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "user_id", Value: 1}}},
				{Keys: bson.D{{Key: "created_at", Value: -1}}},
			},
		},
		{
			collection: "ws_tickets",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "ticket", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
			},
		},
		{
			collection: "refresh_tokens",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "token", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "user_id", Value: 1}}},
				{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
			},
		},
		{
			collection: "crypto_payments",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "payment_id", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "order_id", Value: 1}}}, // Non-unique to allow null values in old records
				{Keys: bson.D{{Key: "user_id", Value: 1}}},
				{Keys: bson.D{{Key: "status", Value: 1}}},
			},
		},
	}
}

// createIndexes creates all required indexes for optimal performance
func createIndexes(ctx context.Context) error {
	for _, def := range getAllIndexes() {
		collection := DB.Collection(def.collection)
		if _, err := collection.Indexes().CreateMany(ctx, def.indexes); err != nil {
			log.Error().Err(err).Str("collection", def.collection).Msg("failed_to_create_indexes")
			return err
		}
	}

	log.Info().Msg("database_indexes_created")
	return nil
}
