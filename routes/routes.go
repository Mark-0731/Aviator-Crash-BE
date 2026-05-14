package routes

import (
	"aviator-backend/controllers"
	"aviator-backend/middleware"

	"github.com/gin-gonic/gin"
)

// Controllers holds all controller instances
type Controllers struct {
	Auth   *controllers.AuthController
	User   *controllers.UserController
	Wallet *controllers.WalletController
	Game   *controllers.GameController
	Admin  *controllers.AdminController
}

// newControllers initializes all controllers
func newControllers() *Controllers {
	return &Controllers{
		Auth:   controllers.NewAuthController(),
		User:   controllers.NewUserController(),
		Wallet: controllers.NewWalletController(),
		Game:   controllers.NewGameController(),
		Admin:  controllers.NewAdminController(),
	}
}

// SetupRoutes configures all application routes
func SetupRoutes(router *gin.Engine) {
	ctrl := newControllers()

	// Health check (no rate limiting)
	router.GET("/health", ctrl.Game.Health)

	// API v1 group
	api := router.Group("/api")
	{
		setupAuthRoutes(api, ctrl)
		setupUserRoutes(api, ctrl)
		setupWalletRoutes(api, ctrl)
		setupGameRoutes(api, ctrl)
		setupAdminRoutes(api, ctrl)
	}
}

// setupAuthRoutes configures authentication routes
func setupAuthRoutes(api *gin.RouterGroup, ctrl *Controllers) {
	auth := api.Group("/auth")
	
	// Apply strict rate limiting (5 req/min) to login/register endpoints
	authStrict := auth.Group("")
	authStrict.Use(middleware.RateLimitAuth())
	{
		authStrict.POST("/register", ctrl.Auth.Register)
		authStrict.POST("/login", ctrl.Auth.Login)
		authStrict.POST("/refresh", ctrl.Auth.RefreshToken)
		authStrict.POST("/logout", ctrl.Auth.Logout)
	}

	// Apply standard REST rate limiting (60 req/min) to ws-ticket
	authStandard := auth.Group("")
	authStandard.Use(middleware.RateLimitREST())
	{
		authStandard.POST("/ws-ticket", middleware.JWTAuth(), ctrl.Auth.CreateWSTicket)
	}
}

// setupUserRoutes configures user routes
func setupUserRoutes(api *gin.RouterGroup, ctrl *Controllers) {
	user := api.Group("/user")
	user.Use(middleware.JWTAuth(), middleware.RateLimitREST())
	{
		user.GET("/profile", ctrl.User.GetProfile)
		user.GET("/bets", ctrl.User.GetBets)
		user.GET("/transactions", ctrl.User.GetTransactions)
		user.PUT("/client-seed", ctrl.User.SetClientSeed) // Provably fair: player-provided seed
	}
}

// setupWalletRoutes configures wallet routes
func setupWalletRoutes(api *gin.RouterGroup, ctrl *Controllers) {
	wallet := api.Group("/wallet")
	wallet.Use(middleware.JWTAuth(), middleware.RateLimitREST())
	{
		wallet.POST("/deposit", ctrl.Wallet.Deposit)
		wallet.GET("/balance", ctrl.Wallet.GetBalance)
	}
}

// setupGameRoutes configures game routes
func setupGameRoutes(api *gin.RouterGroup, ctrl *Controllers) {
	game := api.Group("/game")
	game.Use(middleware.RateLimitREST())
	{
		game.GET("/history", ctrl.Game.GetHistory)
		game.GET("/verify", ctrl.Game.Verify)
		game.GET("/state", ctrl.Game.GetState)
	}
}

// setupAdminRoutes configures admin routes
func setupAdminRoutes(api *gin.RouterGroup, ctrl *Controllers) {
	admin := api.Group("/admin")
	admin.Use(middleware.JWTAuth(), middleware.AdminOnly(), middleware.RateLimitREST())
	{
		admin.GET("/users", ctrl.Admin.GetUsers)
		admin.POST("/users/:id/ban", ctrl.Admin.BanUser)
		admin.POST("/users/:id/unban", ctrl.Admin.UnbanUser)
		admin.POST("/users/:id/adjust-balance", ctrl.Admin.AdjustBalance)
		admin.GET("/rounds", ctrl.Admin.GetRounds)
		admin.GET("/stats", ctrl.Admin.GetStats)
	}
}
