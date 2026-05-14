package controllers

import (
	"aviator-backend/middleware"
	"aviator-backend/services"
	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController() *AuthController {
	return &AuthController{
		authService: services.NewAuthService(),
	}
}

// Register handles user registration
func (ctrl *AuthController) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, err.Error())
		return
	}

	ip := c.ClientIP()
	user, accessToken, refreshToken, err := ctrl.authService.Register(c.Request.Context(), req.Username, req.Email, req.Password, ip)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, err.Error())
		return
	}

	utils.RespondCreated(c, gin.H{
		"token":         accessToken,
		"refresh_token": refreshToken,
		"user":          user.ToResponse(),
	})
}

// Login handles user authentication
func (ctrl *AuthController) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, err.Error())
		return
	}

	ip := c.ClientIP()
	user, accessToken, refreshToken, err := ctrl.authService.Login(c.Request.Context(), req.Email, req.Password, ip)
	if err != nil {
		utils.RespondWithError(c, utils.ErrUnauthorized, err.Error())
		return
	}

	utils.RespondSuccess(c, gin.H{
		"token":         accessToken,
		"refresh_token": refreshToken,
		"user":          user.ToResponse(),
	})
}

// RefreshToken generates a new access token
func (ctrl *AuthController) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, err.Error())
		return
	}

	accessToken, err := ctrl.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		utils.RespondWithError(c, utils.ErrUnauthorized, err.Error())
		return
	}

	utils.RespondSuccess(c, gin.H{
		"token": accessToken,
	})
}

// Logout invalidates a refresh token
func (ctrl *AuthController) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, err.Error())
		return
	}

	if err := ctrl.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, err.Error())
		return
	}

	utils.RespondSuccess(c, gin.H{
		"message": "Logged out successfully",
	})
}

// CreateWSTicket creates a WebSocket authentication ticket
func (ctrl *AuthController) CreateWSTicket(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	ticket, err := ctrl.authService.CreateWSTicket(c.Request.Context(), userID)
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to create ticket")
		return
	}

	utils.RespondSuccess(c, gin.H{
		"ticket": ticket,
	})
}
