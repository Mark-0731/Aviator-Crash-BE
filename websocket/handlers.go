package websocket

import (
	"context"
	"time"

	"aviator-backend/game"
	"aviator-backend/models"
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
		if err == utils.ErrDuplicateBetError {
			client.sendError("DUPLICATE_BET", "You already have a bet in this round")
		} else if err.Error() == "insufficient balance" || err.Error() == "mongo: no documents in result" {
			client.sendError("INSUFFICIENT_BALANCE", "Insufficient balance or account banned")
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

// handleCashOut processes a cash out request using transactional GameService
func handleCashOut(client *Client, data map[string]any) {
	userMutex := getUserMutex(client.userID)
	userMutex.Lock()
	defer userMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gameState := client.hub.gameState
	if gameState == nil {
		client.sendError("INTERNAL_ERROR", "Game state not available")
		return
	}

	phase, err := gameState.GetPhase()
	if err != nil || phase != models.RoundStatusRunning {
		client.sendError("INVALID_PHASE", "Cash out only allowed during running phase")
		return
	}

	currentRound, err := gameState.GetCurrentRound()
	if err != nil {
		client.sendError("INTERNAL_ERROR", "Failed to get current round")
		return
	}

	bet, err := gameState.GetActiveBet(client.userID)
	if err != nil {
		client.sendError("VALIDATION_ERROR", "No active bet found")
		return
	}

	if bet.Status != models.BetStatusPending {
		client.sendError("ALREADY_CASHED_OUT", "Bet already settled")
		return
	}

	engine := client.hub.engine
	if engine == nil {
		client.sendError("INTERNAL_ERROR", "Game engine not available")
		return
	}

	// Use the current multiplier from game state (not recalculated)
	// This ensures the cashout uses the exact multiplier the user saw
	multiplierFloat, err := gameState.GetCurrentMultiplier()
	if err != nil || multiplierFloat < 1.0 {
		client.sendError("INTERNAL_ERROR", "Failed to get current multiplier")
		return
	}

	multiplierX100 := utils.MultiplierToX100(multiplierFloat)

	payoutCents := utils.CalculatePayout(bet.AmountCents, multiplierX100)

	cappedPayout, wasCapped := utils.ApplyMaxWinCap(payoutCents)
	if wasCapped {
		log.Info().
			Str("user_id", client.userID).
			Str("round_id", currentRound.RoundID).
			Int64("original_payout", payoutCents).
			Int64("capped_payout", cappedPayout).
			Msg("max_win_cap_applied")
		payoutCents = cappedPayout
	}

	profitCents := utils.CalculateProfit(bet.AmountCents, payoutCents)

	userIDObj, _ := primitive.ObjectIDFromHex(client.userID)

	// Use GameService for transactional cashout
	// This atomically: updates bet, credits balance, records transaction
	gameService := client.hub.gameService
	if gameService == nil {
		client.sendError("INTERNAL_ERROR", "Game service not available")
		return
	}

	updatedUser, err := gameService.CashOut(ctx, bet.ID, userIDObj, multiplierX100, profitCents, payoutCents, currentRound.RoundID)
	if err != nil {
		log.Error().Err(err).Msg("failed_to_cashout")
		client.sendError("INTERNAL_ERROR", "Failed to process cash out")
		return
	}

	bet.Status = models.BetStatusWon
	bet.CashoutMultiplierX100 = &multiplierX100
	bet.ProfitCents = profitCents
	cashedOutAt := time.Now()
	bet.CashedOutAt = &cashedOutAt

	logCashOut(client.userID, currentRound.RoundID, multiplierFloat, profitCents)

	client.SendMessage("cashout_confirmed", map[string]any{
		"multiplier":    multiplierFloat,
		"profit":        utils.FormatCents(profitCents),
		"profit_cents":  profitCents,
		"payout":        utils.FormatCents(payoutCents),
		"payout_cents":  payoutCents,
		"balance":       utils.FormatCents(updatedUser.BalanceCents),
		"balance_cents": updatedUser.BalanceCents,
	})

	client.hub.Broadcast("player_cashout", map[string]any{
		"username":   client.username,
		"multiplier": multiplierFloat,
		"amount":     utils.FormatCents(payoutCents),
	})
}
