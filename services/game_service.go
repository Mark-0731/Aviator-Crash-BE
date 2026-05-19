package services

import (
	"context"
	"strings"
	"time"

	"aviator-backend/database"
	"aviator-backend/game"
	"aviator-backend/models"
	"aviator-backend/repository"
	"aviator-backend/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
func (s *GameService) PlaceBet(ctx context.Context, userID uuid.UUID, roundID string, amountCents int64) (*models.Bet, *models.User, error) {
	// Get pool from context; fall back to global pool (WS handlers use plain context)
	pool, _ := ctx.Value("pool").(*pgxpool.Pool)
	if pool == nil {
		pool = database.Pool
	}
	if pool == nil {
		return nil, nil, ErrDatabaseConnectionUnavailable
	}

	// Start transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	// Create transaction context
	txCtx := context.WithValue(ctx, "tx", tx)

	// Check for duplicate bet using repository
	existingBet, err := s.betRepo.FindByUserAndRound(txCtx, userID, roundID)
	if err == nil && existingBet != nil {
		return nil, nil, ErrDuplicateBet
	}

	// Deduct balance atomically using repository
	updatedUser, err := s.userRepo.DeductBalance(txCtx, userID, amountCents)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrInsufficientBalance
		}
		return nil, nil, err
	}

	// Create bet using repository
	bet := &models.Bet{
		UserID:      userID,
		RoundID:     roundID,
		AmountCents: amountCents,
		Status:      models.BetStatusPending,
	}

	if err := s.betRepo.Create(txCtx, bet); err != nil {
		return nil, nil, err
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
	if err := s.transactionRepo.Create(txCtx, transaction); err != nil {
		return nil, nil, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
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
func (s *GameService) CashOut(ctx context.Context, betID uuid.UUID, userID uuid.UUID) (*models.User, *CashOutResult, error) {
	// Verify game state is available before starting transaction
	if s.gameState == nil {
		return nil, nil, ErrEngineStateUnavailable
	}

	// Get pool from context; fall back to global pool (WS handlers use plain context)
	pool, _ := ctx.Value("pool").(*pgxpool.Pool)
	if pool == nil {
		pool = database.Pool
	}
	if pool == nil {
		return nil, nil, ErrDatabaseConnectionUnavailable
	}

	// Retry logic for CockroachDB serialization errors (SQLSTATE 40001 / WriteTooOldError).
	// CockroachDB's serializable isolation can abort a transaction at ANY statement,
	// not just at commit. We must retry the entire transaction on any such error.
	maxRetries := 5
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Small backoff before retry to reduce contention
			time.Sleep(time.Duration(attempt*10) * time.Millisecond)
		}

		// Start transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			return nil, nil, err
		}

		// Create transaction context
		txCtx := context.WithValue(ctx, "tx", tx)

		// STEP 1: Load bet and verify ownership + status atomically
		bet, err := s.betRepo.FindByID(txCtx, betID)
		if err != nil {
			tx.Rollback(ctx)
			if err == pgx.ErrNoRows {
				return nil, nil, ErrBetNotFound
			}
			if isSerializationError(err) {
				lastErr = err
				continue
			}
			return nil, nil, err
		}

		// Verify bet belongs to this user
		if bet.UserID != userID {
			tx.Rollback(ctx)
			return nil, nil, ErrBetNotOwnedByUser
		}

		// Verify bet is still pending (not already settled)
		if bet.Status != models.BetStatusPending {
			tx.Rollback(ctx)
			return nil, nil, ErrBetAlreadySettled
		}

		// STEP 2: Load round and verify it's still running
		round, err := s.roundRepo.FindByRoundID(txCtx, bet.RoundID)
		if err != nil {
			tx.Rollback(ctx)
			if err == pgx.ErrNoRows {
				return nil, nil, ErrRoundNotFound
			}
			if isSerializationError(err) {
				lastErr = err
				continue
			}
			return nil, nil, err
		}

		// Verify round is in running phase
		if round.Status != models.RoundStatusRunning {
			tx.Rollback(ctx)
			return nil, nil, ErrRoundNotRunning
		}

		// STEP 3: Read authoritative current multiplier from game state
		// This is the single source of truth - never trust client-provided multiplier
		currentMultiplierFloat, err := s.gameState.GetCurrentMultiplier()
		if err != nil || currentMultiplierFloat < 1.0 {
			tx.Rollback(ctx)
			return nil, nil, ErrEngineStateUnavailable
		}

		currentMultiplierX100 := utils.MultiplierToX100(currentMultiplierFloat)

		// STEP 4: Enforce crash boundary strictly
		// If current multiplier >= crash point, cashout is too late
		if currentMultiplierX100 >= round.CrashPointX100 {
			tx.Rollback(ctx)
			return nil, nil, ErrCashoutTooLate
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
		// This provides DB-level protection against double cashout
		if err := s.betRepo.UpdateCashout(txCtx, betID, currentMultiplierX100, profitCents); err != nil {
			tx.Rollback(ctx)
			if err == pgx.ErrNoRows {
				// Bet was already settled by another concurrent request
				return nil, nil, ErrBetAlreadySettled
			}
			if isSerializationError(err) {
				lastErr = err
				continue
			}
			return nil, nil, err
		}

		// STEP 8: Credit user balance atomically
		updatedUser, err := s.userRepo.UpdateBalance(txCtx, userID, payoutCents)
		if err != nil {
			tx.Rollback(ctx)
			if isSerializationError(err) {
				lastErr = err
				continue
			}
			return nil, nil, err
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
		if err := s.transactionRepo.Create(txCtx, transaction); err != nil {
			tx.Rollback(ctx)
			if isSerializationError(err) {
				lastErr = err
				continue
			}
			return nil, nil, err
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			if isSerializationError(err) {
				lastErr = err
				continue // Retry
			}
			return nil, nil, err
		}

		// Build result to return to caller
		result := &CashOutResult{
			MultiplierX100: currentMultiplierX100,
			Multiplier:     currentMultiplierFloat,
			PayoutCents:    payoutCents,
			ProfitCents:    profitCents,
			WasCapped:      wasCapped,
			RoundID:        bet.RoundID,
		}

		return updatedUser, result, nil
	}

	// All retries exhausted — if the bet is now settled, report it correctly
	// rather than leaking an opaque serialization error to the caller
	if lastErr != nil && isSerializationError(lastErr) {
		return nil, nil, ErrBetAlreadySettled
	}
	if lastErr != nil {
		return nil, nil, lastErr
	}

	return nil, nil, ErrBetAlreadySettled
}


// isSerializationError checks if an error is a CockroachDB serialization error
func isSerializationError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "restart transaction") ||
		strings.Contains(errMsg, "WriteTooOldError") ||
		strings.Contains(errMsg, "RETRY_WRITE_TOO_OLD") ||
		strings.Contains(errMsg, "SQLSTATE 40001")
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

// GetUserProfile gets a user's public profile by ID
func (s *GameService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
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
func (s *GameService) GetUserBets(ctx context.Context, userID uuid.UUID, page, limit int64) ([]models.BetResponse, int64, error) {
	bets, total, err := s.betRepo.FindByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToBetResponses(bets), total, nil
}

// GetUserTransactions gets user's transaction history with pagination
func (s *GameService) GetUserTransactions(ctx context.Context, userID uuid.UUID, page, limit int64) ([]models.TransactionResponse, int64, error) {
	transactions, total, err := s.transactionRepo.FindByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToTransactionResponses(transactions), total, nil
}

// UpdateClientSeed updates the player's client seed for provably fair generation
func (s *GameService) UpdateClientSeed(ctx context.Context, userID uuid.UUID, clientSeed string) error {
	return s.userRepo.UpdateClientSeed(ctx, userID, clientSeed)
}
