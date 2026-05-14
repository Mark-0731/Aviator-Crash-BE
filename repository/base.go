package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BaseRepository provides common database operations
type BaseRepository struct {
	collection *mongo.Collection
}

// newBase creates a new base repository
func newBase(collection *mongo.Collection) *BaseRepository {
	return &BaseRepository{collection: collection}
}

// FindByID finds a document by ID
func (r *BaseRepository) FindByID(ctx context.Context, id primitive.ObjectID, result interface{}) error {
	return r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(result)
}

// FindOne finds a single document by filter
func (r *BaseRepository) FindOne(ctx context.Context, filter interface{}, result interface{}) error {
	return r.collection.FindOne(ctx, filter).Decode(result)
}

// FindAll finds all documents matching filter
func (r *BaseRepository) FindAll(ctx context.Context, filter interface{}, results interface{}) error {
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	return cursor.All(ctx, results)
}

// FindWithPagination finds documents with pagination
func (r *BaseRepository) FindWithPagination(ctx context.Context, filter interface{}, results interface{}, page, limit int64, sortField string, sortOrder int) (int64, error) {
	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	// Get paginated results
	skip := (page - 1) * limit
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		return 0, err
	}

	return total, nil
}

// InsertOne inserts a single document
func (r *BaseRepository) InsertOne(ctx context.Context, document interface{}) (primitive.ObjectID, error) {
	result, err := r.collection.InsertOne(ctx, document)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return result.InsertedID.(primitive.ObjectID), nil
}

// UpdateOne updates a single document
func (r *BaseRepository) UpdateOne(ctx context.Context, filter, update interface{}) error {
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// UpdateMany updates multiple documents
func (r *BaseRepository) UpdateMany(ctx context.Context, filter, update interface{}) (int64, error) {
	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

// DeleteOne deletes a single document
func (r *BaseRepository) DeleteOne(ctx context.Context, filter interface{}) error {
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// DeleteMany deletes multiple documents
func (r *BaseRepository) DeleteMany(ctx context.Context, filter interface{}) (int64, error) {
	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// CountDocuments counts documents matching filter
func (r *BaseRepository) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return r.collection.CountDocuments(ctx, filter)
}

// Aggregate performs aggregation pipeline
func (r *BaseRepository) Aggregate(ctx context.Context, pipeline interface{}, results interface{}) error {
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	return cursor.All(ctx, results)
}

// DeleteExpired deletes documents where expires_at < now
func (r *BaseRepository) DeleteExpired(ctx context.Context) (int64, error) {
	return r.DeleteMany(ctx, bson.D{
		{Key: "expires_at", Value: bson.D{{Key: "$lt", Value: time.Now()}}},
	})
}
