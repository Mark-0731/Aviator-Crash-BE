package websocket

import (
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
)

// getUserMutex gets or creates a mutex for a specific user
func getUserMutex(userID string) *sync.Mutex {
	mu, _ := userMutexes.LoadOrStore(userID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// cleanupUserMutex removes a user's mutex when they disconnect
func cleanupUserMutex(userID string) {
	userMutexes.Delete(userID)
}

// marshalMessage marshals a message to JSON
func marshalMessage(event string, data any) ([]byte, error) {
	message := map[string]any{
		"event": event,
		"data":  data,
	}
	return json.Marshal(message)
}

// sendToChannel attempts to send a message to a channel without blocking
func sendToChannel(ch chan []byte, msg []byte, userID string) bool {
	select {
	case ch <- msg:
		return true
	default:
		log.Warn().Str("user_id", userID).Msg("send_buffer_full")
		return false
	}
}

// logBetPlaced logs a bet placement event
func logBetPlaced(userID, roundID string, amountCents int64) {
	log.Info().
		Str("user_id", userID).
		Str("round_id", roundID).
		Int64("amount_cents", amountCents).
		Msg("bet_placed")
}

// logCashOut logs a cash out event
func logCashOut(userID, roundID string, multiplier float64, profitCents int64) {
	log.Info().
		Str("user_id", userID).
		Str("round_id", roundID).
		Float64("multiplier", multiplier).
		Int64("profit_cents", profitCents).
		Msg("player_cashed_out")
}

// logPlayerConnection logs player connection events
func logPlayerConnection(userID, username, action string) {
	log.Info().
		Str("user_id", userID).
		Str("username", username).
		Msg(action)
}
