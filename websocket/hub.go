package websocket

import (
	"sync"
	"time"

	"aviator-backend/game"
	"aviator-backend/services"

	"github.com/rs/zerolog/log"
)

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Game state reference (using concrete interface from game package)
	gameState game.GameStateStore

	// Game engine reference (using concrete type)
	engine *game.Engine

	// Game service for transactional operations
	gameService *services.GameService

	// Mutex for thread-safe client map access
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub(gameState game.GameStateStore, gameService *services.GameService) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte, 256),
		Register:    make(chan *Client),
		unregister:  make(chan *Client),
		gameState:   gameState,
		gameService: gameService,
	}
}

// SetEngine sets the game engine reference (called after engine creation to avoid circular init)
func (h *Hub) SetEngine(engine *game.Engine) {
	h.engine = engine
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			logPlayerConnection(client.userID, client.username, "player_connected")
			go h.sendGameSync(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				logPlayerConnection(client.userID, client.username, "player_disconnected")
			}
			h.mu.Unlock()

			// Clean up per-user mutex when client disconnects
			cleanupUserMutex(client.userID)

		case message := <-h.broadcast:
			// Read-lock to iterate; collect slow clients
			h.mu.RLock()
			var slowClients []*Client
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					slowClients = append(slowClients, client)
				}
			}
			h.mu.RUnlock()

			// Write-lock to remove slow clients safely
			if len(slowClients) > 0 {
				h.mu.Lock()
				for _, client := range slowClients {
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						close(client.send)
						log.Warn().Str("user_id", client.userID).Msg("slow_client_disconnected")
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(event string, data any) {
	jsonMessage, err := marshalMessage(event, data)
	if err != nil {
		log.Error().Err(err).Msg("failed_to_marshal_broadcast_message")
		return
	}
	h.broadcast <- jsonMessage
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(client *Client, event string, data any) {
	jsonMessage, err := marshalMessage(event, data)
	if err != nil {
		log.Error().Err(err).Msg("failed_to_marshal_client_message")
		return
	}

	select {
	case client.send <- jsonMessage:
	default:
		log.Warn().Str("user_id", client.userID).Msg("client_send_buffer_full")
	}
}

// GetConnectedCount returns the number of connected clients
func (h *Hub) GetConnectedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetConnectedUserIDs returns the userIDs of all connected clients (for client seed collection)
func (h *Hub) GetConnectedUserIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.clients))
	for client := range h.clients {
		ids = append(ids, client.userID)
	}
	return ids
}

// sendGameSync sends current game state to a newly connected client
func (h *Hub) sendGameSync(client *Client) {
	if h.gameState == nil {
		return
	}

	phase, err := h.gameState.GetPhase()
	if err != nil {
		log.Error().Err(err).Msg("failed_to_get_phase_for_sync")
		return
	}

	currentRound, err := h.gameState.GetCurrentRound()
	if err != nil {
		log.Error().Err(err).Msg("failed_to_get_round_for_sync")
		return
	}

	playerCount, _ := h.gameState.GetPlayerCount()

	syncData := map[string]any{
		"phase":            phase,
		"round_id":         currentRound.RoundID,
		"server_seed_hash": currentRound.ServerSeedHash,
		"player_count":     playerCount,
	}

	if phase == "running" && h.engine != nil {
		elapsed := time.Since(h.engine.GetRoundStartTime()).Seconds()
		multiplier := game.CalculateCurrentMultiplier(elapsed)
		syncData["multiplier"] = multiplier
	}

	activeBet, err := h.gameState.GetActiveBet(client.userID)
	if err == nil && activeBet != nil {
		syncData["active_bet"] = map[string]any{
			"amount_cents": activeBet.AmountCents,
			"status":       activeBet.Status,
		}
	}

	h.SendToClient(client, "game_sync", syncData)

	log.Info().
		Str("user_id", client.userID).
		Str("phase", string(phase)).
		Str("round_id", currentRound.RoundID).
		Msg("game_sync_sent")
}
