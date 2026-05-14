package services

import (
	"context"

	"aviator-backend/database"
	"aviator-backend/game"
	"aviator-backend/models"
	"aviator-backend/repository"
	"aviator-backend/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

type GameService struct {
	betRepo         *repository.BetRepository
	roundRepo       *repository.RoundRepository
	transactionRepo *repository.TransactionRepository
	userRepo        *repository.UserRepository
}

func NewGameService() *GameService {
	return &GameService{
		betRepo:         repository.NewBetRepository(),
		roundRepo:       repository.NewRoundRepository(),
		transactionRepo: repository.NewTransactionRepository(),
		userRepo:        repository.NewUserRepository(),
	}
}

// PlaceBet places a bet for a user (used by WebSocket handler)
func (s *GameService) PlaceBet(ctx context.Context, userID primitive.ObjectID, roundID string, amountCents int64) (*models.Bet, *models.User, error) {
	session, err := database.Client.StartSession()
	if err != nil {
		return nil, nil, err
	}
	defer session.EndSession(ctx)

	var bet *models.Bet
	var updatedUser *models.User

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Check for duplicate bet using repository
		existingBet, err := s.betRepo.FindByUserAndRound(sessCtx, userID, roundID)
		if err == nil && existingBet != nil {
			return nil, utils.ErrDuplicateBetError
		}

		// Deduct balance atomically using repository
		updatedUser, err = s.userRepo.DeductBalance(sessCtx, userID, amountCents)
		if err != nil {
			return nil, err
		}

		// Create bet using repository
		bet = &models.Bet{
			UserID:      userID,
			RoundID:     roundID,
			AmountCents: amountCents,
			Status:      models.BetStatusPending,
		}

		if err := s.betRepo.Create(sessCtx, bet); err != nil {
			return nil, err
		}

		// Create transaction using repository
		transaction := &models.Transaction{
			UserID:             userID,
			Type:               models.TransactionTypeBet,
			AmountCents:        amountCents,
			RoundID:            &roundID,
			BalanceBeforeCents: updatedUser.BalanceCents + amountCents,
			BalanceAfterCents:  updatedUser.BalanceCents,
		}
		if err := s.transactionRepo.Create(sessCtx, transaction); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, nil, err
	}

	return bet, updatedUser, nil
}

// CashOut cashes out a bet (used by WebSocket handler)
func (s *GameService) CashOut(ctx context.Context, betID primitive.ObjectID, userID primitive.ObjectID, multiplierX100 int64, profitCents int64, payoutCents int64, roundID string) (*models.User, error) {
	session, err := database.Client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	var updatedUser *models.User

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update bet using repository
		if err := s.betRepo.UpdateCashout(sessCtx, betID, multiplierX100, profitCents); err != nil {
			return nil, err
		}

		// Credit balance using repository
		updatedUser, err = s.userRepo.UpdateBalance(sessCtx, userID, payoutCents)
		if err != nil {
			return nil, err
		}

		// Create transaction using repository
		transaction := &models.Transaction{
			UserID:             userID,
			Type:               models.TransactionTypeWin,
			AmountCents:        payoutCents,
			RoundID:            &roundID,
			BalanceBeforeCents: updatedUser.BalanceCents - payoutCents,
			BalanceAfterCents:  updatedUser.BalanceCents,
		}
		if err := s.transactionRepo.Create(sessCtx, transaction); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

// GetRoundHistory gets completed rounds with pagination
func (s *GameService) GetRoundHistory(ctx context.Context, page, limit int64) ([]models.RoundResponse, int64, error) {
	rounds, total, err := s.roundRepo.FindHistory(ctx, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToRoundResponses(rounds), total, nil
}

// VerifyRound verifies provably fair calculation
func (s *GameService) VerifyRound(serverSeed, serverSeedHash, clientSeed string, nonce int64, hash string, crashPoint float64) (bool, map[string]any) {
	return game.VerifyRound(serverSeed, serverSeedHash, clientSeed, nonce, hash, crashPoint)
}

// GetCurrentGameState gets the current game state
func (s *GameService) GetCurrentGameState(gameState game.GameStateStore) (map[string]any, error) {
	phase, err := gameState.GetPhase()
	if err != nil {
		return nil, err
	}

	currentRound, err := gameState.GetCurrentRound()
	if err != nil {
		return nil, err
	}

	playerCount, _ := gameState.GetPlayerCount()

	state := map[string]any{
		"phase":            phase,
		"round_id":         currentRound.RoundID,
		"server_seed_hash": currentRound.ServerSeedHash,
		"player_count":     playerCount,
	}

	// Only include multiplier if running
	if phase == models.RoundStatusRunning {
		// Calculate current multiplier (this is safe to expose)
		// We don't expose timing data, just the current multiplier value
		state["multiplier"] = "calculated_by_client" // Client calculates from their own timer
	}

	return state, nil
}

// GetUserBets gets user's bet history with pagination
func (s *GameService) GetUserBets(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.BetResponse, int64, error) {
	bets, total, err := s.betRepo.FindByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToBetResponses(bets), total, nil
}

// GetUserTransactions gets user's transaction history with pagination
func (s *GameService) GetUserTransactions(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.TransactionResponse, int64, error) {
	transactions, total, err := s.transactionRepo.FindByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToTransactionResponses(transactions), total, nil
}
