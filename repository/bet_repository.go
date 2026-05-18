package repository

import (
	"context"
	"time"

	"aviator-backend/database"
	"aviator-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BetRepository struct {
	*BaseRepository
}

func NewBetRepository() *BetRepository {
	return &BetRepository{
		BaseRepository: newBase(database.DB.Collection("bets")),
	}
}

func (r *BetRepository) Create(ctx context.Context, bet *models.Bet) error {
	bet.PlacedAt = time.Now()
	id, err := r.InsertOne(ctx, bet)
	if err != nil {
		return err
	}
	bet.ID = id
	return nil
}

func (r *BetRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Bet, error) {
	var bet models.Bet
	err := r.BaseRepository.FindByID(ctx, id, &bet)
	return &bet, err
}

func (r *BetRepository) FindByUserAndRound(ctx context.Context, userID primitive.ObjectID, roundID string) (*models.Bet, error) {
	var bet models.Bet
	err := r.FindOne(ctx, bson.D{
		{Key: "user_id", Value: userID},
		{Key: "round_id", Value: roundID},
	}, &bet)
	return &bet, err
}

func (r *BetRepository) FindPendingByRound(ctx context.Context, roundID string) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.FindAll(ctx, bson.D{
		{Key: "round_id", Value: roundID},
		{Key: "status", Value: models.BetStatusPending},
	}, &bets)
	return bets, err
}

func (r *BetRepository) FindAllPending(ctx context.Context) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.FindAll(ctx, bson.D{{Key: "status", Value: models.BetStatusPending}}, &bets)
	return bets, err
}

func (r *BetRepository) UpdateCashout(ctx context.Context, betID primitive.ObjectID, multiplierX100 int64, profitCents int64) error {
	// Conditional update: only update if status is still pending
	// This provides DB-level protection against double cashout
	result, err := r.collection.UpdateOne(ctx,
		bson.D{
			{Key: "_id", Value: betID},
			{Key: "status", Value: models.BetStatusPending}, // ← Condition: only if still pending
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "cashout_multiplier_x100", Value: multiplierX100},
			{Key: "profit_cents", Value: profitCents},
			{Key: "status", Value: models.BetStatusWon},
			{Key: "cashed_out_at", Value: time.Now()},
		}}},
	)

	if err != nil {
		return err
	}

	// If no document was matched, bet was already settled
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments // Caller will interpret as ErrBetAlreadySettled
	}

	return nil
}

func (r *BetRepository) UpdateStatus(ctx context.Context, betID primitive.ObjectID, status models.BetStatus, profitCents int64) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: betID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "profit_cents", Value: profitCents},
		}}},
	)
}

func (r *BetRepository) BulkUpdateLost(ctx context.Context, bets []models.Bet) error {
	if len(bets) == 0 {
		return nil
	}

	bulkModels := make([]mongo.WriteModel, 0, len(bets))
	for _, bet := range bets {
		bulkModels = append(bulkModels, mongo.NewUpdateOneModel().
			SetFilter(bson.D{{Key: "_id", Value: bet.ID}}).
			SetUpdate(bson.D{{Key: "$set", Value: bson.D{
				{Key: "status", Value: models.BetStatusLost},
				{Key: "profit_cents", Value: -bet.AmountCents},
			}}}))
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := r.collection.BulkWrite(ctx, bulkModels, opts)
	return err
}

func (r *BetRepository) FindWonByRound(ctx context.Context, roundID string) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.FindAll(ctx, bson.D{
		{Key: "round_id", Value: roundID},
		{Key: "status", Value: models.BetStatusWon},
	}, &bets)
	return bets, err
}

func (r *BetRepository) FindByUser(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Bet, int64, error) {
	var bets []models.Bet
	total, err := r.FindWithPagination(ctx, bson.D{{Key: "user_id", Value: userID}}, &bets, page, limit, "placed_at", -1)
	return bets, total, err
}

func (r *BetRepository) FindByRound(ctx context.Context, roundID string) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.FindAll(ctx, bson.D{{Key: "round_id", Value: roundID}}, &bets)
	return bets, err
}

func (r *BetRepository) CountByRound(ctx context.Context, roundID string) (int64, error) {
	return r.CountDocuments(ctx, bson.D{{Key: "round_id", Value: roundID}})
}

func (r *BetRepository) GetTotalWageredByRound(ctx context.Context, roundID string) (int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "round_id", Value: roundID}}}},
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
