package game

import (
	"context"
	"fmt"
	"math"
	"time"

	"aviator-backend/config"
	"aviator-backend/models"
	"aviator-backend/utils"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// waitingPhase handles the waiting state
func (e *Engine) waitingPhase() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	round := e.generateNewRound()

	// Save round with retry
	if err := e.saveRoundWithRetry(ctx, round); err != nil {
		log.Error().Err(err).Str("round_id", round.RoundID).Msg("failed_to_insert_round_after_retries")
		return
	}

	// Update state
	e.mu.Lock()
	e.currentRound = round
	e.mu.Unlock()

	e.state.SetPhase(models.RoundStatusWaiting)
	e.state.SetCurrentRound(round)
	e.state.ClearActiveBets()

	log.Info().
		Str("round_id", round.RoundID).
		Str("phase", string(models.RoundStatusWaiting)).
		Float64("crash_point", float64(round.CrashPointX100)/100.0).
		Msg("round_started")

	// Broadcast round start (NEVER reveal crash_point or server_seed)
	e.hub.Broadcast("round_start", map[string]interface{}{
		"round_id":            round.RoundID,
		"server_seed_hash":    round.ServerSeedHash,
		"waiting_duration_ms": config.AppConfig.WaitingDuration.Milliseconds(),
	})

	time.Sleep(config.AppConfig.WaitingDuration)

	// Record transition time for grace window
	e.mu.Lock()
	round.TransitionTime = time.Now()
	e.mu.Unlock()

	e.state.SetCurrentRound(round)
}

// runningPhase handles the running state
func (e *Engine) runningPhase() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	round, err := e.state.GetCurrentRound()
	if err != nil {
		log.Error().Err(err).Msg("failed_to_get_current_round")
		return
	}

	e.state.SetPhase(models.RoundStatusRunning)

	e.mu.Lock()
	e.roundStartTime = time.Now()
	e.mu.Unlock()

	// Calculate player stats
	activeBets, _ := e.state.GetActiveBets()
	playerCount := len(activeBets)
	totalWagered := calculateTotalWagered(activeBets)

	round.PlayerCount = playerCount
	round.TotalWageredCents = totalWagered

	// Write the modified copy back to state so crashedPhase sees updated values
	e.state.SetCurrentRound(round)

	// Update round in database
	e.updateRoundRunningWithRetry(ctx, round.RoundID, playerCount, totalWagered)

	log.Info().
		Str("round_id", round.RoundID).
		Int("player_count", playerCount).
		Int64("total_wagered_cents", totalWagered).
		Msg("round_running")

	// Broadcast that multiplier is now running
	e.hub.Broadcast("round_running", map[string]interface{}{
		"round_id":      round.RoundID,
		"player_count":  playerCount,
		"total_wagered": totalWagered,
	})

	// Run multiplier loop
	e.runMultiplierLoop(round.CrashPointX100)
}

// crashedPhase handles the crashed state
func (e *Engine) crashedPhase() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	round, err := e.state.GetCurrentRound()
	if err != nil {
		log.Error().Err(err).Msg("failed_to_get_current_round")
		return
	}

	e.state.SetPhase(models.RoundStatusCrashed)
	crashedAt := time.Now()
	round.CrashedAt = &crashedAt

	// Settle bets
	totalPayout := e.settleBets(ctx, round)
	round.TotalPayoutCents = totalPayout

	// Write the modified copy back to state
	e.state.SetCurrentRound(round)

	// Update round
	if err := e.roundRepo.UpdateCrashed(ctx, round.RoundID, totalPayout); err != nil {
		log.Error().Err(err).Msg("failed_to_update_round")
	}

	// Update consecutive crashes counter
	crashPoint := float64(round.CrashPointX100) / 100.0
	e.mu.Lock()
	if IsInstantCrash(crashPoint) {
		e.consecutiveInstantCrashes++
	} else {
		e.consecutiveInstantCrashes = 0
	}
	e.mu.Unlock()

	log.Info().
		Str("round_id", round.RoundID).
		Float64("crash_point", crashPoint).
		Int64("total_wagered", round.TotalWageredCents).
		Int64("total_payout", totalPayout).
		Int64("house_profit", round.TotalWageredCents-totalPayout).
		Msg("round_crashed")

	// Generate next round's server seed NOW (pre-commitment for chain verification)
	nextServerSeed := uuid.New().String()
	nextServerSeedHash := utils.SHA256Hash(nextServerSeed)
	// Store next seed for the upcoming round
	e.mu.Lock()
	e.nextServerSeed = nextServerSeed
	e.mu.Unlock()

	// Broadcast crash (NOW reveal server_seed and crash_point)
	// Also include next_server_seed_hash so players can verify the chain
	e.hub.Broadcast("round_crash", map[string]interface{}{
		"crash_point":           crashPoint,
		"round_id":              round.RoundID,
		"server_seed":           round.ServerSeed,
		"client_seed":           round.ClientSeed,
		"nonce":                 round.Nonce,
		"hash":                  round.Hash,
		"next_server_seed_hash": nextServerSeedHash, // Pre-commitment for next round
	})

	time.Sleep(config.AppConfig.CrashDuration)
}

// runMultiplierLoop runs the multiplier growth loop
func (e *Engine) runMultiplierLoop(crashPointX100 int64) {
	lastBroadcastTime := time.Now()
	minBroadcastInterval := 50 * time.Millisecond

	for {
		e.mu.RLock()
		elapsed := time.Since(e.roundStartTime).Seconds()
		e.mu.RUnlock()

		// Calculate multiplier using exponential growth
		// Formula: e^(0.00006 * elapsed_milliseconds)
		multiplierFloat := math.Exp(0.00006 * elapsed * 1000)
		multiplierX100 := int64(math.Floor(multiplierFloat * 100))

		// Store current multiplier in game state for cashouts
		e.state.SetCurrentMultiplier(multiplierFloat)

		// Stop when we reach or exceed crash point
		if multiplierX100 >= crashPointX100 {
			break
		}

		// Throttled broadcasting
		if time.Since(lastBroadcastTime) >= minBroadcastInterval {
			e.hub.Broadcast("multiplier_update", map[string]interface{}{
				"multiplier": float64(multiplierX100) / 100.0,
			})
			lastBroadcastTime = time.Now()
		}

		// Jittered tick interval
		e.mu.RLock()
		jitter := e.rng.Intn(config.AppConfig.MultiplierTickMaxMS - config.AppConfig.MultiplierTickMinMS)
		e.mu.RUnlock()

		time.Sleep(time.Duration(config.AppConfig.MultiplierTickMinMS+jitter) * time.Millisecond)
	}

	// Final broadcast at crash point
	finalMultiplier := float64(crashPointX100) / 100.0
	e.state.SetCurrentMultiplier(finalMultiplier)
	e.hub.Broadcast("multiplier_update", map[string]interface{}{
		"multiplier": finalMultiplier,
	})
}

// generateNewRound creates a new round with provably fair parameters.
// The crash point is the raw cryptographic result — never overridden.
// This matches Stake/BC.game: no artificial floors or minimums.
func (e *Engine) generateNewRound() *models.Round {
	roundID := uuid.New().String()
	nonce := time.Now().Unix()

	// Use pre-committed server seed if available, otherwise generate a new one
	e.mu.Lock()
	serverSeed := e.nextServerSeed
	e.nextServerSeed = ""
	e.mu.Unlock()
	if serverSeed == "" {
		serverSeed = uuid.New().String()
	}

	// Combine all connected players' client seeds
	clientSeed := e.buildCombinedClientSeed(nonce)

	// Pure cryptographic crash point — no overrides, fully verifiable
	crashPoint := GenerateCrashPoint(serverSeed, clientSeed, nonce)

	return &models.Round{
		ID:             primitive.NewObjectID(),
		RoundID:        roundID,
		CrashPointX100: utils.MultiplierToX100(crashPoint),
		ServerSeed:     serverSeed,
		ServerSeedHash: utils.SHA256Hash(serverSeed),
		ClientSeed:     clientSeed,
		Nonce:          nonce,
		Hash:           GenerateHash(serverSeed, clientSeed, nonce),
		Status:         models.RoundStatusWaiting,
		StartedAt:      time.Now(),
	}
}

// buildCombinedClientSeed fetches all connected players' client seeds from DB
// and combines them via SHA256 to form a single unpredictable client seed.
func (e *Engine) buildCombinedClientSeed(nonce int64) string {
	userIDs := e.hub.GetConnectedUserIDs()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Convert string IDs to ObjectIDs
	objectIDs := make([]interface{}, 0, len(userIDs))
	for _, id := range userIDs {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			objectIDs = append(objectIDs, oid)
		}
	}

	// Use a default if no players are connected
	if len(objectIDs) == 0 {
		return fmt.Sprintf("server-generated:%d", nonce)
	}

	// Convert back to primitive.ObjectID slice for repo
	oidSlice := make([]primitive.ObjectID, 0, len(objectIDs))
	for _, id := range objectIDs {
		oidSlice = append(oidSlice, id.(primitive.ObjectID))
	}

	seeds, err := e.userRepo.GetClientSeedsByIDs(ctx, oidSlice)
	if err != nil || len(seeds) == 0 {
		return fmt.Sprintf("server-generated:%d", nonce)
	}

	// Combine all seeds deterministically via SHA256
	combined := fmt.Sprintf("%d:%s", nonce, joinSeeds(seeds))
	return utils.SHA256Hash(combined)
}

// joinSeeds joins seeds with a separator
func joinSeeds(seeds []string) string {
	result := ""
	for i, s := range seeds {
		if i > 0 {
			result += ":"
		}
		result += s
	}
	return result
}

// Helper functions

func calculateTotalWagered(bets map[string]*models.Bet) int64 {
	total := int64(0)
	for _, bet := range bets {
		total += bet.AmountCents
	}
	return total
}

func (e *Engine) saveRoundWithRetry(ctx context.Context, round *models.Round) error {
	for attempt := 0; attempt < MaxRetries; attempt++ {
		if err := e.roundRepo.Create(ctx, round); err == nil {
			return nil
		} else if attempt < MaxRetries-1 {
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Str("round_id", round.RoundID).
				Msg("failed_to_insert_round_retrying")
			time.Sleep(RetryDelay * time.Duration(attempt+1))
		} else {
			return err
		}
	}
	return nil
}

func (e *Engine) updateRoundRunningWithRetry(ctx context.Context, roundID string, playerCount int, totalWagered int64) {
	for attempt := 0; attempt < MaxRetries; attempt++ {
		if err := e.roundRepo.UpdateRunning(ctx, roundID, playerCount, totalWagered); err == nil {
			return
		} else if attempt < MaxRetries-1 {
			time.Sleep(RetryDelay)
		} else {
			log.Error().Err(err).Msg("failed_to_update_round_running")
		}
	}
}
