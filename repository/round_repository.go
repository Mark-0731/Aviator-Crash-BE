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

type RoundRepository struct {
	*BaseRepository
}

func NewRoundRepository() *RoundRepository {
	return &RoundRepository{
		BaseRepository: newBase(database.DB.Collection("rounds")),
	}
}

func (r *RoundRepository) Create(ctx context.Context, round *models.Round) error {
	round.StartedAt = time.Now()
	id, err := r.InsertOne(ctx, round)
	if err != nil {
		return err
	}
	round.ID = id
	return nil
}

func (r *RoundRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Round, error) {
	var round models.Round
	err := r.BaseRepository.FindByID(ctx, id, &round)
	return &round, err
}

func (r *RoundRepository) FindByRoundID(ctx context.Context, roundID string) (*models.Round, error) {
	var round models.Round
	err := r.FindOne(ctx, bson.D{{Key: "round_id", Value: roundID}}, &round)
	return &round, err
}

func (r *RoundRepository) UpdateStatus(ctx context.Context, roundID string, status models.RoundStatus) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "round_id", Value: roundID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}}}},
	)
}

func (r *RoundRepository) UpdateRunning(ctx context.Context, roundID string, playerCount int, totalWageredCents int64) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "round_id", Value: roundID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: models.RoundStatusRunning},
			{Key: "player_count", Value: playerCount},
			{Key: "total_wagered_cents", Value: totalWageredCents},
		}}},
	)
}

func (r *RoundRepository) UpdateCrashed(ctx context.Context, roundID string, totalPayoutCents int64) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "round_id", Value: roundID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: models.RoundStatusCrashed},
			{Key: "crashed_at", Value: time.Now()},
			{Key: "total_payout_cents", Value: totalPayoutCents},
		}}},
	)
}

func (r *RoundRepository) FindHistory(ctx context.Context, page, limit int64) ([]models.Round, int64, error) {
	var rounds []models.Round
	total, err := r.FindWithPagination(ctx,
		bson.D{{Key: "status", Value: models.RoundStatusCrashed}},
		&rounds, page, limit, "started_at", -1)
	return rounds, total, err
}

func (r *RoundRepository) FindAll(ctx context.Context, page, limit int64) ([]models.Round, int64, error) {
	var rounds []models.Round
	total, err := r.FindWithPagination(ctx, bson.D{}, &rounds, page, limit, "started_at", -1)
	return rounds, total, err
}

func (r *RoundRepository) GetLatestRound(ctx context.Context) (*models.Round, error) {
	var rounds []models.Round
	_, err := r.FindWithPagination(ctx, bson.D{}, &rounds, 1, 1, "started_at", -1)
	if err != nil || len(rounds) == 0 {
		return nil, err
	}
	return &rounds[0], nil
}

func (r *RoundRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "status", Value: models.RoundStatusCrashed}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total_rounds", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "total_wagered", Value: bson.D{{Key: "$sum", Value: "$total_wagered_cents"}}},
			{Key: "total_payout", Value: bson.D{{Key: "$sum", Value: "$total_payout_cents"}}},
		}}},
	}

	var results []map[string]interface{}
	if err := r.Aggregate(ctx, pipeline, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return map[string]interface{}{
			"total_rounds":  0,
			"total_wagered": 0,
			"total_payout":  0,
			"house_profit":  0,
		}, nil
	}

	result := results[0]
	totalWagered := result["total_wagered"].(int64)
	totalPayout := result["total_payout"].(int64)
	result["house_profit"] = totalWagered - totalPayout

	return result, nil
}

func (r *RoundRepository) CountByStatus(ctx context.Context, status models.RoundStatus) (int64, error) {
	return r.CountDocuments(ctx, bson.D{{Key: "status", Value: status}})
}
