package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"aviator-backend/middleware"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// Per-user mutex to prevent race conditions on concurrent bets
var userMutexes sync.Map

const (
	// Time allowed to write a message to the peer
	WriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	PongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than PongWait)
	PingPeriod = 50 * time.Second

	// Maximum message size allowed from peer (4 KB — sufficient for all client messages)
	MaxMessageSize = 4096
)

// Client represents a WebSocket client connection
type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	userID      string
	username    string
	isAdmin     bool
	rateLimiter *middleware.WSRateLimiter
}

// ClientMessage represents an incoming message from client
type ClientMessage struct {
	Event string         `json:"event"`
	Data  map[string]any `json:"data"`
}

// NewClient creates a new Client instance
func NewClient(hub *Hub, conn *websocket.Conn, userID, username string, isAdmin bool) *Client {
	return &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256), // Buffered channel
		userID:      userID,
		username:    username,
		isAdmin:     isAdmin,
		rateLimiter: middleware.NewWSRateLimiter(),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("user_id", c.userID).Msg("ws_read_error")
			}
			break
		}

		// Parse message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Warn().Err(err).Str("user_id", c.userID).Msg("invalid_message_format")
			c.sendError("VALIDATION_ERROR", "Invalid message format")
			continue
		}

		// Check per-connection rate limit
		if !c.rateLimiter.Allow() {
			c.sendError("RATE_LIMITED", "Too many messages, please slow down")
			continue
		}

		// Handle message
		HandleClientMessage(c, &clientMsg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(code, message string) {
	errorMsg, err := marshalMessage("error", map[string]any{
		"code":    code,
		"message": message,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed_to_marshal_error")
		return
	}
	sendToChannel(c.send, errorMsg, c.userID)
}

// SendMessage sends a message to this client
func (c *Client) SendMessage(event string, data any) {
	jsonMsg, err := marshalMessage(event, data)
	if err != nil {
		log.Error().Err(err).Msg("failed_to_marshal_message")
		return
	}
	sendToChannel(c.send, jsonMsg, c.userID)
}
