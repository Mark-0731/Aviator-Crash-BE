package controllers

import (
	"aviator-backend/middleware"
	"aviator-backend/repository"
	"aviator-backend/services"
	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

type UserController struct {
	gameService *services.GameService
	userRepo    *repository.UserRepository
}

func NewUserController() *UserController {
	return &UserController{
		gameService: services.NewGameService(),
		userRepo:    repository.NewUserRepository(),
	}
}

// GetProfile gets user profile
func (ctrl *UserController) GetProfile(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseObjectID(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	user, err := ctrl.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		utils.RespondWithError(c, utils.ErrNotFound, "User not found")
		return
	}

	utils.RespondSuccess(c, user.ToResponse())
}

// GetBets gets user's bet history
func (ctrl *UserController) GetBets(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseObjectID(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	page, limit := parsePagination(c, 10)

	bets, total, err := ctrl.gameService.GetUserBets(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to fetch bets")
		return
	}

	utils.RespondSuccess(c, paginationResponse(bets, total, page, limit))
}

// GetTransactions gets user's transaction history
func (ctrl *UserController) GetTransactions(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseObjectID(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	page, limit := parsePagination(c, 10)

	transactions, total, err := ctrl.gameService.GetUserTransactions(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to fetch transactions")
		return
	}

	response := paginationResponse(transactions, total, page, limit)
	response["transactions"] = response["data"]
	delete(response, "data")
	utils.RespondSuccess(c, response)
}

// SetClientSeed lets a player set their own client seed for provably fair play
// This seed will be combined with other players' seeds when generating the next round
func (ctrl *UserController) SetClientSeed(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseObjectID(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		ClientSeed string `json:"client_seed" binding:"required,min=1,max=128"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "client_seed is required (1-128 chars)")
		return
	}

	if err := ctrl.userRepo.UpdateClientSeed(c.Request.Context(), userID, req.ClientSeed); err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to update client seed")
		return
	}

	utils.RespondSuccess(c, gin.H{
		"message":     "Client seed updated",
		"client_seed": req.ClientSeed,
	})
}
