package websocket

import (
	"net/http"

	"aviator-backend/config"
	"aviator-backend/services"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// Handler handles WebSocket upgrade requests
type Handler struct {
	hub         *Hub
	authService *services.AuthService
	upgrader    gorillaws.Upgrader
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub:         hub,
		authService: services.NewAuthService(),
		upgrader: gorillaws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				for _, allowed := range config.AppConfig.AllowedOrigins {
					if origin == allowed {
						return true
					}
				}
				return false
			},
		},
	}
}

// HandleConnection handles WebSocket connection upgrade
func (h *Handler) HandleConnection(c *gin.Context) {
	// Get ticket from query parameter
	ticket := c.Query("ticket")
	if ticket == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ticket required"})
		return
	}

	// Validate ticket
	user, err := h.authService.ValidateWSTicket(c.Request.Context(), ticket)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("websocket_upgrade_failed")
		return
	}

	// Create client
	client := NewClient(h.hub, conn, user.ID.String(), user.Username, user.IsAdmin)

	// Register client
	h.hub.Register <- client

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}
