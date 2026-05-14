package repository

import (
	"context"
	"time"

	"aviator-backend/database"
	"aviator-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionRepository struct {
	*BaseRepository
}

func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{
		BaseRepository: newBase(database.DB.Collection("transactions")),
	}
}

func (r *TransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	transaction.CreatedAt = time.Now()
	id, err := r.InsertOne(ctx, transaction)
	if err != nil {
		return err
	}
	transaction.ID = id
	return nil
}

func (r *TransactionRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.BaseRepository.FindByID(ctx, id, &transaction)
	return &transaction, err
}

func (r *TransactionRepository) FindByUser(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	total, err := r.FindWithPagination(ctx,
		bson.D{{Key: "user_id", Value: userID}},
		&transactions, page, limit, "created_at", -1)
	return transactions, total, err
}

func (r *TransactionRepository) FindByRound(ctx context.Context, roundID string) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.FindAll(ctx, bson.D{{Key: "round_id", Value: roundID}}, &transactions)
	return transactions, err
}

func (r *TransactionRepository) FindByType(ctx context.Context, transactionType models.TransactionType, page, limit int64) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	total, err := r.FindWithPagination(ctx,
		bson.D{{Key: "type", Value: transactionType}},
		&transactions, page, limit, "created_at", -1)
	return transactions, total, err
}

func (r *TransactionRepository) GetTotalByType(ctx context.Context, transactionType models.TransactionType) (int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "type", Value: transactionType}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount_cents"}}},
		}}},
	}

	var result []struct {
		Total int64 `bson:"total"`
	}

	if err := r.Aggregate(ctx, pipeline, &result); err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}

	return result[0].Total, nil
}

func (r *TransactionRepository) GetUserStats(ctx context.Context, userID primitive.ObjectID) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "user_id", Value: userID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$type"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount_cents"}}},
		}}},
	}

	var results []map[string]interface{}
	if err := r.Aggregate(ctx, pipeline, &results); err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	for _, result := range results {
		transactionType := result["_id"].(string)
		stats[transactionType] = map[string]interface{}{
			"count": result["count"],
			"total": result["total"],
		}
	}

	return stats, nil
}
