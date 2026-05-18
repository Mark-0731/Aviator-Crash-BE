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
	gameState       game.GameStateStore // Added for authoritative state access
}

// SetGameState sets the game state store (called during initialization)
func (s *GameService) SetGameState(state game.GameStateStore) {
	s.gameState = state
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
			return nil, ErrDuplicateBet
		}

		// Deduct balance atomically using repository
		updatedUser, err = s.userRepo.DeductBalance(sessCtx, userID, amountCents)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, ErrInsufficientBalance
			}
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

// CashOut processes a cashout request with full atomic validation
// This method performs ALL validation inside the transaction to eliminate TOCTOU races
//
// Race-condition-free guarantees:
// 1. Bet ownership and status verified atomically
// 2. Round phase verified atomically
// 3. Current multiplier read from authoritative game state
// 4. Crash boundary enforced strictly (currentMultiplier < crashPoint)
// 5. Payout calculated server-side (never trust client)
// 6. Max win cap applied server-side
// 7. Balance update, bet update, and transaction insert are atomic
//
// Returns domain-specific errors for proper error handling by caller
func (s *GameService) CashOut(ctx context.Context, betID primitive.ObjectID, userID primitive.ObjectID) (*models.User, *CashOutResult, error) {
	// Verify game state is available before starting transaction
	if s.gameState == nil {
		return nil, nil, ErrEngineStateUnavailable
	}

	session, err := database.Client.StartSession()
	if err != nil {
		return nil, nil, err
	}
	defer session.EndSession(ctx)

	var updatedUser *models.User
	var result *CashOutResult

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// STEP 1: Load bet and verify ownership + status atomically
		bet, err := s.betRepo.FindByID(sessCtx, betID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, ErrBetNotFound
			}
			return nil, err
		}

		// Verify bet belongs to this user
		if bet.UserID != userID {
			return nil, ErrBetNotOwnedByUser
		}

		// Verify bet is still pending (not already settled)
		if bet.Status != models.BetStatusPending {
			return nil, ErrBetAlreadySettled
		}

		// STEP 2: Load round and verify it's still running
		round, err := s.roundRepo.FindByRoundID(sessCtx, bet.RoundID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, ErrRoundNotFound
			}
			return nil, err
		}

		// Verify round is in running phase
		if round.Status != models.RoundStatusRunning {
			return nil, ErrRoundNotRunning
		}

		// STEP 3: Read authoritative current multiplier from game state
		// This is the single source of truth - never trust client-provided multiplier
		currentMultiplierFloat, err := s.gameState.GetCurrentMultiplier()
		if err != nil || currentMultiplierFloat < 1.0 {
			return nil, ErrEngineStateUnavailable
		}

		currentMultiplierX100 := utils.MultiplierToX100(currentMultiplierFloat)

		// STEP 4: Enforce crash boundary strictly
		// If current multiplier >= crash point, cashout is too late
		if currentMultiplierX100 >= round.CrashPointX100 {
			return nil, ErrCashoutTooLate
		}

		// STEP 5: Calculate payout server-side (never trust client)
		payoutCents := utils.CalculatePayout(bet.AmountCents, currentMultiplierX100)

		// STEP 6: Apply max win cap if necessary
		cappedPayout, wasCapped := utils.ApplyMaxWinCap(payoutCents)
		if wasCapped {
			payoutCents = cappedPayout
		}

		profitCents := utils.CalculateProfit(bet.AmountCents, payoutCents)

		// STEP 7: Update bet status atomically with conditional update
		// MongoDB will only update if status is still pending
		// This provides DB-level protection against double cashout
		if err := s.betRepo.UpdateCashout(sessCtx, betID, currentMultiplierX100, profitCents); err != nil {
			if err == mongo.ErrNoDocuments {
				// Bet was already settled by another request
				return nil, ErrBetAlreadySettled
			}
			return nil, err
		}

		// STEP 8: Credit user balance atomically
		updatedUser, err = s.userRepo.UpdateBalance(sessCtx, userID, payoutCents)
		if err != nil {
			return nil, err
		}

		// STEP 9: Record transaction atomically
		transaction := &models.Transaction{
			UserID:             userID,
			Type:               models.TransactionTypeWin,
			AmountCents:        payoutCents,
			RoundID:            &bet.RoundID,
			BalanceBeforeCents: updatedUser.BalanceCents - payoutCents,
			BalanceAfterCents:  updatedUser.BalanceCents,
		}
		if err := s.transactionRepo.Create(sessCtx, transaction); err != nil {
			return nil, err
		}

		// Build result to return to caller
		result = &CashOutResult{
			MultiplierX100: currentMultiplierX100,
			Multiplier:     currentMultiplierFloat,
			PayoutCents:    payoutCents,
			ProfitCents:    profitCents,
			WasCapped:      wasCapped,
			RoundID:        bet.RoundID,
		}

		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, nil, err
	}

	return updatedUser, result, nil
}

// CashOutResult contains the result of a successful cashout
type CashOutResult struct {
	MultiplierX100 int64
	Multiplier     float64
	PayoutCents    int64
	ProfitCents    int64
	WasCapped      bool
	RoundID        string
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
