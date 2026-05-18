package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aviator-backend/config"
	"aviator-backend/database"
	"aviator-backend/game"
	"aviator-backend/middleware"
	"aviator-backend/routes"
	"aviator-backend/services"
	"aviator-backend/websocket"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize logger
	initLogger()

	// Load configuration
	if err := config.Load(); err != nil {
		log.Fatal().Err(err).Msg("failed_to_load_config")
	}

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatal().Err(err).Msg("failed_to_connect_database")
	}
	defer database.Disconnect()

	// Initialize game components
	// Order matters: create hub first (with nil engine), then engine with hub, then set engine on hub
	gameState := game.NewInMemoryGameState()
	gameService := services.NewGameService()
	gameService.SetGameState(gameState) // Set game state for atomic cashout validation
	wsHub := websocket.NewHub(gameState, gameService)
	gameEngine := game.NewEngine(gameState, wsHub)
	wsHub.SetEngine(gameEngine)

	// Start services
	go wsHub.Run()
	gameEngine.Start()

	// Start cleanup service
	cleanupService := services.NewCleanupService()
	cleanupService.Start()

	// Setup HTTP server
	router := setupRouter(wsHub)
	srv := &http.Server{
		Addr:    ":" + config.AppConfig.Port,
		Handler: router,
	}

	// Start server
	go startServer(srv)

	// Wait for shutdown signal
	waitForShutdown(srv, gameEngine, cleanupService)
}

// initLogger initializes the zerolog logger
func initLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if config.AppConfig == nil || config.AppConfig.Env != "production" {
		// Development: pretty colored console output
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		// Production: JSON output, only warn/error level logs
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
}

// setupRouter configures the Gin router
func setupRouter(wsHub *websocket.Hub) *gin.Engine {
	if config.AppConfig.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(middleware.CORS())

	// Setup routes
	routes.SetupRoutes(router)

	// WebSocket endpoint
	wsHandler := websocket.NewHandler(wsHub)
	router.GET("/ws/game", wsHandler.HandleConnection)

	return router
}

// startServer starts the HTTP server
func startServer(srv *http.Server) {
	log.Info().
		Str("port", config.AppConfig.Port).
		Str("env", config.AppConfig.Env).
		Msg("server_started")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server_failed")
	}
}

// waitForShutdown waits for interrupt signal and performs graceful shutdown
func waitForShutdown(srv *http.Server, gameEngine *game.Engine, cleanupService *services.CleanupService) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting_down_server")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server_forced_shutdown")
	}

	// Stop() is safe to call multiple times now (uses sync.Once)
	gameEngine.Stop()
	cleanupService.Stop()

	log.Info().Msg("server_stopped")
}
