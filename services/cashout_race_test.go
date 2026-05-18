package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"aviator-backend/config"
	"aviator-backend/database"
	"aviator-backend/game"
	"aviator-backend/models"
	"aviator-backend/repository"
	"aviator-backend/utils"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Test 1: Spam same cashout 20 times in parallel for one user
// Expected: Only one should succeed
func TestConcurrentCashoutPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	setupTestDB(t)
	defer cleanupTestDB(t)

	gameState := game.NewInMemoryGameState()
	gameService := NewGameService()
	gameService.SetGameState(gameState)

	userRepo := repository.NewUserRepository()
	betRepo := repository.NewBetRepository()

	// Create test user with balance
	user := createTestUser(t, ctx, userRepo, 100000) // $1000 balance

	// Create test round
	round := createTestRound(t, ctx, gameState, 500) // Crash at 5.00x

	// Place bet
	betAmount := int64(10000) // $100 bet
	bet, _, err := gameService.PlaceBet(ctx, user.ID, round.RoundID, betAmount)
	require.NoError(t, err)

	// Add bet to game state
	gameState.AddActiveBet(user.ID.Hex(), bet)

	// Set round to running in both game state AND database
	setRoundRunning(t, ctx, gameState, round, 1, betAmount)

	// Set multiplier to 2.50x (safe to cash out)
	gameState.SetCurrentMultiplier(2.50)

	// Spam cashout 20 times in parallel
	numAttempts := 20
	var wg sync.WaitGroup
	results := make(chan error, numAttempts)

	t.Logf("Starting %d concurrent cashout attempts for user %s", numAttempts, user.ID.Hex())

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func(attempt int) {
			defer wg.Done()
			_, _, err := gameService.CashOut(ctx, bet.ID, user.ID)
			results <- err
			if err == nil {
				t.Logf("Attempt %d: SUCCESS", attempt)
			} else {
				t.Logf("Attempt %d: FAILED - %v", attempt, err)
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Count successes and failures
	successes := 0
	alreadySettled := 0
	otherErrors := 0

	for err := range results {
		if err == nil {
			successes++
		} else if err == ErrBetAlreadySettled {
			alreadySettled++
		} else {
			otherErrors++
			t.Logf("Unexpected error: %v", err)
		}
	}

	t.Logf("Results: %d successes, %d already settled, %d other errors",
		successes, alreadySettled, otherErrors)

	// CRITICAL: Only 1 should succeed
	assert.Equal(t, 1, successes, "Expected exactly 1 successful cashout")
	assert.Equal(t, numAttempts-1, alreadySettled, "Expected all other attempts to fail with ErrBetAlreadySettled")
	assert.Equal(t, 0, otherErrors, "Expected no other errors")

	// Verify bet status in DB
	finalBet, err := betRepo.FindByID(ctx, bet.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BetStatusWon, finalBet.Status, "Bet should be marked as won")
	assert.NotNil(t, finalBet.CashoutMultiplierX100, "Cashout multiplier should be set")

	// Verify user balance
	finalUser, err := userRepo.FindByID(ctx, user.ID)
	require.NoError(t, err)

	expectedPayout := utils.CalculatePayout(betAmount, 250) // 2.50x
	expectedBalance := int64(100000) - betAmount + expectedPayout
	assert.Equal(t, expectedBalance, finalUser.BalanceCents,
		"User balance should reflect exactly one payout")

	t.Logf("✅ Test passed: Only 1 cashout succeeded out of %d attempts", numAttempts)
}

// Test 2: Trigger cashout very near crash boundary
// Expected: No payout should happen when multiplier is equal to or above crash point
func TestCrashBoundaryEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	setupTestDB(t)
	defer cleanupTestDB(t)

	gameState := game.NewInMemoryGameState()
	gameService := NewGameService()
	gameService.SetGameState(gameState)

	userRepo := repository.NewUserRepository()
	betRepo := repository.NewBetRepository()

	// Create test user
	user := createTestUser(t, ctx, userRepo, 100000)

	// Create test round with crash at 5.00x
	crashPointX100 := int64(500) // 5.00x
	round := createTestRound(t, ctx, gameState, crashPointX100)

	// Place bet
	betAmount := int64(10000)
	bet, _, err := gameService.PlaceBet(ctx, user.ID, round.RoundID, betAmount)
	require.NoError(t, err)

	gameState.AddActiveBet(user.ID.Hex(), bet)
	setRoundRunning(t, ctx, gameState, round, 1, betAmount)

	// Test scenarios
	testCases := []struct {
		name          string
		multiplier    float64
		shouldSucceed bool
		expectedError error
	}{
		{
			name:          "Below crash point (4.99x)",
			multiplier:    4.99,
			shouldSucceed: true,
		},
		{
			name:          "Exactly at crash point (5.00x)",
			multiplier:    5.00,
			shouldSucceed: false,
			expectedError: ErrCashoutTooLate,
		},
		{
			name:          "Above crash point (5.01x)",
			multiplier:    5.01,
			shouldSucceed: false,
			expectedError: ErrCashoutTooLate,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For subsequent tests after the first, create new round and bet
			if i > 0 {
				// Create new round
				round = createTestRound(t, ctx, gameState, crashPointX100)

				// Place new bet
				bet, _, err = gameService.PlaceBet(ctx, user.ID, round.RoundID, betAmount)
				require.NoError(t, err)

				gameState.AddActiveBet(user.ID.Hex(), bet)
				setRoundRunning(t, ctx, gameState, round, 1, betAmount)
			}

			// Set multiplier
			gameState.SetCurrentMultiplier(tc.multiplier)

			// Attempt cashout
			_, result, err := gameService.CashOut(ctx, bet.ID, user.ID)

			if tc.shouldSucceed {
				assert.NoError(t, err, "Cashout should succeed below crash point")
				assert.NotNil(t, result, "Result should be returned")
				t.Logf("✅ Cashout succeeded at %.2fx (below crash point)", tc.multiplier)
			} else {
				assert.Error(t, err, "Cashout should fail at or above crash point")
				assert.Equal(t, tc.expectedError, err, "Should return ErrCashoutTooLate")
				assert.Nil(t, result, "Result should be nil on error")

				// Verify bet is still pending (not settled)
				finalBet, err := betRepo.FindByID(ctx, bet.ID)
				require.NoError(t, err)
				assert.Equal(t, models.BetStatusPending, finalBet.Status,
					"Bet should remain pending after failed cashout")

				t.Logf("✅ Cashout correctly rejected at %.2fx (at/above crash point)", tc.multiplier)
			}
		})
	}
}

// Test 3: Force round crash while cashout request is mid-flight
// Expected: Request should fail
func TestMidFlightCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	setupTestDB(t)
	defer cleanupTestDB(t)

	gameState := game.NewInMemoryGameState()
	gameService := NewGameService()
	gameService.SetGameState(gameState)

	userRepo := repository.NewUserRepository()
	roundRepo := repository.NewRoundRepository()

	// Create test user
	user := createTestUser(t, ctx, userRepo, 100000)

	// Create test round
	round := createTestRound(t, ctx, gameState, 500)

	// Place bet
	betAmount := int64(10000)
	bet, _, err := gameService.PlaceBet(ctx, user.ID, round.RoundID, betAmount)
	require.NoError(t, err)

	gameState.AddActiveBet(user.ID.Hex(), bet)
	setRoundRunning(t, ctx, gameState, round, 1, betAmount)
	gameState.SetCurrentMultiplier(3.50)

	// Start cashout in goroutine with delay
	cashoutDone := make(chan error)
	go func() {
		// Simulate network delay
		time.Sleep(10 * time.Millisecond)
		_, _, err := gameService.CashOut(ctx, bet.ID, user.ID)
		cashoutDone <- err
	}()

	// Crash round immediately (before cashout completes)
	time.Sleep(5 * time.Millisecond)

	// Update round status to crashed in DB
	err = roundRepo.UpdateCrashed(ctx, round.RoundID, 0)
	require.NoError(t, err)

	// Update game state
	gameState.SetPhase(models.RoundStatusCrashed)

	// Wait for cashout result
	cashoutErr := <-cashoutDone

	// Should fail with either ErrRoundNotRunning or ErrCashoutTooLate
	assert.Error(t, cashoutErr, "Cashout should fail when round crashes mid-flight")
	assert.True(t,
		cashoutErr == ErrRoundNotRunning || cashoutErr == ErrCashoutTooLate,
		"Expected ErrRoundNotRunning or ErrCashoutTooLate, got: %v", cashoutErr)

	t.Logf("✅ Mid-flight cashout correctly failed with: %v", cashoutErr)
}

// Test 4: Check balance, bet status, and ledger entry always match after cashout
// Expected: No partial updates
func TestDataConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	setupTestDB(t)
	defer cleanupTestDB(t)

	gameState := game.NewInMemoryGameState()
	gameService := NewGameService()
	gameService.SetGameState(gameState)

	userRepo := repository.NewUserRepository()
	betRepo := repository.NewBetRepository()
	transactionRepo := repository.NewTransactionRepository()

	// Create test user
	initialBalance := int64(100000) // $1000
	user := createTestUser(t, ctx, userRepo, initialBalance)

	// Create test round
	round := createTestRound(t, ctx, gameState, 500)

	// Place bet
	betAmount := int64(10000) // $100
	bet, userAfterBet, err := gameService.PlaceBet(ctx, user.ID, round.RoundID, betAmount)
	require.NoError(t, err)

	balanceAfterBet := userAfterBet.BalanceCents
	assert.Equal(t, initialBalance-betAmount, balanceAfterBet,
		"Balance should be reduced by bet amount")

	gameState.AddActiveBet(user.ID.Hex(), bet)
	setRoundRunning(t, ctx, gameState, round, 1, betAmount)

	// Set multiplier to 2.50x
	multiplier := 2.50
	gameState.SetCurrentMultiplier(multiplier)

	// Cashout
	userAfterCashout, result, err := gameService.CashOut(ctx, bet.ID, user.ID)
	require.NoError(t, err, "Cashout should succeed")
	require.NotNil(t, result, "Result should not be nil")

	// Calculate expected values
	expectedMultiplierX100 := int64(250) // 2.50x
	expectedPayoutCents := utils.CalculatePayout(betAmount, expectedMultiplierX100)
	expectedProfitCents := utils.CalculateProfit(betAmount, expectedPayoutCents)
	expectedFinalBalance := balanceAfterBet + expectedPayoutCents

	t.Logf("Expected: multiplier=%d, payout=%d, profit=%d, balance=%d",
		expectedMultiplierX100, expectedPayoutCents, expectedProfitCents, expectedFinalBalance)

	// VERIFY 1: Bet status and values
	finalBet, err := betRepo.FindByID(ctx, bet.ID)
	require.NoError(t, err)

	assert.Equal(t, models.BetStatusWon, finalBet.Status,
		"Bet status should be 'won'")
	assert.NotNil(t, finalBet.CashoutMultiplierX100,
		"Cashout multiplier should be set")
	assert.Equal(t, expectedMultiplierX100, *finalBet.CashoutMultiplierX100,
		"Cashout multiplier should match")
	assert.Equal(t, expectedProfitCents, finalBet.ProfitCents,
		"Profit cents should match")
	assert.NotNil(t, finalBet.CashedOutAt,
		"CashedOutAt timestamp should be set")

	t.Logf("✅ Bet data consistent: status=%s, multiplier=%d, profit=%d",
		finalBet.Status, *finalBet.CashoutMultiplierX100, finalBet.ProfitCents)

	// VERIFY 2: User balance
	finalUser, err := userRepo.FindByID(ctx, user.ID)
	require.NoError(t, err)

	assert.Equal(t, expectedFinalBalance, finalUser.BalanceCents,
		"User balance should match expected")
	assert.Equal(t, expectedFinalBalance, userAfterCashout.BalanceCents,
		"Returned user balance should match DB")

	t.Logf("✅ Balance consistent: expected=%d, actual=%d",
		expectedFinalBalance, finalUser.BalanceCents)

	// VERIFY 3: Transaction record
	transactions, _, err := transactionRepo.FindByUser(ctx, user.ID, 1, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(transactions), 2,
		"Should have at least 2 transactions (bet + win)")

	// Find the win transaction (most recent)
	var winTxn *models.Transaction
	for i := range transactions {
		if transactions[i].Type == models.TransactionTypeWin {
			winTxn = &transactions[i]
			break
		}
	}

	require.NotNil(t, winTxn, "Win transaction should exist")
	assert.Equal(t, models.TransactionTypeWin, winTxn.Type,
		"Transaction type should be 'win'")
	assert.Equal(t, expectedPayoutCents, winTxn.AmountCents,
		"Transaction amount should match payout")
	assert.Equal(t, balanceAfterBet, winTxn.BalanceBeforeCents,
		"Balance before should match")
	assert.Equal(t, expectedFinalBalance, winTxn.BalanceAfterCents,
		"Balance after should match")
	assert.Equal(t, round.RoundID, *winTxn.RoundID,
		"Round ID should match")

	t.Logf("✅ Transaction consistent: type=%s, amount=%d, balance_before=%d, balance_after=%d",
		winTxn.Type, winTxn.AmountCents, winTxn.BalanceBeforeCents, winTxn.BalanceAfterCents)

	// VERIFY 4: All values match across bet, balance, and transaction
	assert.Equal(t, finalBet.ProfitCents, expectedProfitCents,
		"Bet profit should match calculated")
	assert.Equal(t, winTxn.AmountCents, expectedPayoutCents,
		"Transaction amount should match calculated payout")
	assert.Equal(t, finalUser.BalanceCents, winTxn.BalanceAfterCents,
		"User balance should match transaction balance after")

	t.Logf("✅ All data consistent: No partial updates detected")
}

// Helper functions

func setupTestDB(t *testing.T) {
	// Load .env from parent directory (since tests run from services/)
	if err := godotenv.Load("../.env"); err != nil {
		t.Logf("Warning: Could not load ../.env file: %v", err)
	}

	// Load config
	if err := config.Load(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	t.Logf("Connecting to MongoDB: %s", config.AppConfig.MongoURI)

	// Connect to test database
	if err := database.Connect(); err != nil {
		t.Skipf("Skipping test: MongoDB not available - %v\n\nTo run these tests:\n1. Ensure MongoDB Atlas is accessible\n2. Check MONGO_URI in .env\n3. Verify network connectivity", err)
	}
}

func cleanupTestDB(t *testing.T) {
	ctx := context.Background()

	// Clean up test data
	database.DB.Collection("users").Drop(ctx)
	database.DB.Collection("bets").Drop(ctx)
	database.DB.Collection("rounds").Drop(ctx)
	database.DB.Collection("transactions").Drop(ctx)

	database.Disconnect()
}

func createTestUser(t *testing.T, ctx context.Context, userRepo *repository.UserRepository, balanceCents int64) *models.User {
	user := &models.User{
		Username:     "testuser_" + uuid.New().String()[:8],
		Email:        "test_" + uuid.New().String()[:8] + "@example.com",
		Password:     "dummy_hash",
		BalanceCents: balanceCents,
		ClientSeed:   uuid.New().String(),
	}

	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	return user
}

func createTestRound(t *testing.T, ctx context.Context, gameState game.GameStateStore, crashPointX100 int64) *models.Round {
	roundID := uuid.New().String()
	serverSeed := uuid.New().String()
	clientSeed := uuid.New().String()
	nonce := time.Now().Unix()

	round := &models.Round{
		ID:             primitive.NewObjectID(),
		RoundID:        roundID,
		CrashPointX100: crashPointX100,
		ServerSeed:     serverSeed,
		ServerSeedHash: utils.SHA256Hash(serverSeed),
		ClientSeed:     clientSeed,
		Nonce:          nonce,
		Hash:           game.GenerateHash(serverSeed, clientSeed, nonce),
		Status:         models.RoundStatusWaiting,
		StartedAt:      time.Now(),
	}

	roundRepo := repository.NewRoundRepository()
	err := roundRepo.Create(ctx, round)
	require.NoError(t, err)

	// Set in game state
	gameState.SetCurrentRound(round)
	gameState.SetPhase(models.RoundStatusWaiting)

	return round
}

// Helper to set round to running state (both in-memory and DB)
func setRoundRunning(t *testing.T, ctx context.Context, gameState game.GameStateStore, round *models.Round, playerCount int, totalWagered int64) {
	gameState.SetPhase(models.RoundStatusRunning)

	roundRepo := repository.NewRoundRepository()
	err := roundRepo.UpdateRunning(ctx, round.RoundID, playerCount, totalWagered)
	require.NoError(t, err)
}
