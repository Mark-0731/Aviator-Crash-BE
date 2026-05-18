package websocket

import (
	"context"
	"time"

	"aviator-backend/game"
	"aviator-backend/models"
	"aviator-backend/services"
	"aviator-backend/utils"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

// HandleClientMessage routes incoming WebSocket messages to appropriate handlers
func HandleClientMessage(client *Client, msg *ClientMessage) {
	switch msg.Event {
	case "place_bet":
		handlePlaceBet(client, msg.Data)
	case "cash_out":
		handleCashOut(client, msg.Data)
	default:
		client.sendError("VALIDATION_ERROR", "Unknown event type")
	}
}

// handlePlaceBet processes a bet placement request using transactional GameService
func handlePlaceBet(client *Client, data map[string]any) {
	userMutex := getUserMutex(client.userID)
	userMutex.Lock()
	defer userMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	amountFloat, ok := data["amount"].(float64)
	if !ok {
		client.sendError("VALIDATION_ERROR", "Invalid amount format")
		return
	}

	amountCents := utils.ToCents(amountFloat)

	if err := utils.ValidateAmount(amountCents); err != nil {
		client.sendError("INVALID_AMOUNT", err.Error())
		return
	}

	gameState := client.hub.gameState
	if gameState == nil {
		client.sendError("INTERNAL_ERROR", "Game state not available")
		return
	}

	phase, err := gameState.GetPhase()
	if err != nil {
		client.sendError("INTERNAL_ERROR", "Failed to get game phase")
		return
	}

	currentRound, err := gameState.GetCurrentRound()
	if err != nil {
		client.sendError("INTERNAL_ERROR", "Failed to get current round")
		return
	}

	now := time.Now()
	isWaitingPhase := phase == models.RoundStatusWaiting
	isWithinGrace := phase == models.RoundStatusRunning &&
		now.Before(currentRound.TransitionTime.Add(game.BetGraceWindow))

	if !isWaitingPhase && !isWithinGrace {
		client.sendError("INVALID_PHASE", "Bets only allowed during waiting phase")
		return
	}

	userIDObj, _ := primitive.ObjectIDFromHex(client.userID)

	// Use GameService for transactional bet placement
	// This atomically: checks duplicate, deducts balance, creates bet, records transaction
	gameService := client.hub.gameService
	if gameService == nil {
		client.sendError("INTERNAL_ERROR", "Game service not available")
		return
	}

	bet, updatedUser, err := gameService.PlaceBet(ctx, userIDObj, currentRound.RoundID, amountCents)
	if err != nil {
		if err == services.ErrDuplicateBet {
			client.sendError("DUPLICATE_BET", "You already have a bet in this round")
		} else if err == services.ErrInsufficientBalance {
			client.sendError("INSUFFICIENT_BALANCE", "Insufficient balance")
		} else {
			log.Error().Err(err).Str("user_id", client.userID).Msg("failed_to_place_bet")
			client.sendError("INTERNAL_ERROR", "Failed to process bet")
		}
		return
	}

	gameState.AddActiveBet(client.userID, bet)

	logBetPlaced(client.userID, currentRound.RoundID, amountCents)

	client.SendMessage("bet_confirmed", map[string]any{
		"amount":        utils.FormatCents(amountCents),
		"amount_cents":  amountCents,
		"balance":       utils.FormatCents(updatedUser.BalanceCents),
		"balance_cents": updatedUser.BalanceCents,
		"round_id":      currentRound.RoundID,
	})

	client.hub.Broadcast("player_bet", map[string]any{
		"username":     client.username,
		"amount":       utils.FormatCents(amountCents),
		"amount_cents": amountCents,
	})
}

// handleCashOut processes a cash out request with minimal pre-checks
// All authoritative validation happens inside GameService.CashOut transaction
//
// Race-condition-free design:
// 1. Acquire per-user mutex (prevents duplicate cashout attempts)
// 2. Call GameService.CashOut with just betID and userID
// 3. GameService performs ALL validation atomically in transaction
// 4. Map domain errors to user-friendly messages
//
// IMPORTANT: We do NOT read phase, round, bet, or multiplier here
// All of that happens atomically inside the transaction
func handleCashOut(client *Client, data map[string]any) {
	// STEP 1: Acquire per-user mutex to prevent concurrent cashout attempts
	userMutex := getUserMutex(client.userID)
	userMutex.Lock()
	defer userMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// STEP 2: Get bet ID from game state (minimal pre-check)
	// We still need to know which bet to cash out
	gameState := client.hub.gameState
	if gameState == nil {
		client.sendError("INTERNAL_ERROR", "Game state not available")
		return
	}

	bet, err := gameState.GetActiveBet(client.userID)
	if err != nil {
		client.sendError("NO_ACTIVE_BET", "No active bet found")
		return
	}

	// STEP 3: Call GameService.CashOut - all validation happens here atomically
	gameService := client.hub.gameService
	if gameService == nil {
		client.sendError("INTERNAL_ERROR", "Game service not available")
		return
	}

	userIDObj, _ := primitive.ObjectIDFromHex(client.userID)

	// This call does ALL validation atomically:
	// - Verifies bet ownership and status
	// - Verifies round is running
	// - Reads authoritative multiplier
	// - Enforces crash boundary
	// - Calculates payout server-side
	// - Applies max win cap
	// - Updates bet, balance, and transaction atomically
	updatedUser, result, err := gameService.CashOut(ctx, bet.ID, userIDObj)

	// STEP 4: Handle domain-specific errors with user-friendly messages
	if err != nil {
		switch err {
		case services.ErrBetNotFound:
			client.sendError("BET_NOT_FOUND", "Bet not found")
		case services.ErrBetNotOwnedByUser:
			client.sendError("UNAUTHORIZED", "Bet does not belong to you")
		case services.ErrBetAlreadySettled:
			client.sendError("ALREADY_SETTLED", "Bet already cashed out or lost")
		case services.ErrRoundNotFound:
			client.sendError("ROUND_NOT_FOUND", "Round not found")
		case services.ErrRoundNotRunning:
			client.sendError("INVALID_PHASE", "Cash out only allowed during running phase")
		case services.ErrCashoutTooLate:
			client.sendError("TOO_LATE", "Multiplier has reached crash point")
		case services.ErrEngineStateUnavailable:
			client.sendError("INTERNAL_ERROR", "Game engine state unavailable")
		default:
			log.Error().Err(err).Str("user_id", client.userID).Msg("cashout_failed")
			client.sendError("INTERNAL_ERROR", "Failed to process cash out")
		}
		return
	}

	// STEP 5: Update local bet state (for broadcast purposes)
	bet.Status = models.BetStatusWon
	bet.CashoutMultiplierX100 = &result.MultiplierX100
	bet.ProfitCents = result.ProfitCents
	cashedOutAt := time.Now()
	bet.CashedOutAt = &cashedOutAt

	// STEP 6: Log successful cashout
	logCashOut(client.userID, result.RoundID, result.Multiplier, result.ProfitCents)

	// STEP 7: Send success response to user
	client.SendMessage("cashout_confirmed", map[string]any{
		"multiplier":    result.Multiplier,
		"profit":        utils.FormatCents(result.ProfitCents),
		"profit_cents":  result.ProfitCents,
		"payout":        utils.FormatCents(result.PayoutCents),
		"payout_cents":  result.PayoutCents,
		"balance":       utils.FormatCents(updatedUser.BalanceCents),
		"balance_cents": updatedUser.BalanceCents,
		"was_capped":    result.WasCapped,
	})

	// STEP 8: Broadcast cashout to all players
	client.hub.Broadcast("player_cashout", map[string]any{
		"username":   client.username,
		"multiplier": result.Multiplier,
		"amount":     utils.FormatCents(result.PayoutCents),
	})
}
