package controllers

import (
	"aviator-backend/middleware"
	"aviator-backend/services"
	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS
// NOTE: Deposit uses MOCK payment - needs real payment gateway in production

type WalletController struct {
	walletService *services.WalletService
}

func NewWalletController() *WalletController {
	return &WalletController{
		walletService: services.NewWalletService(),
	}
}

// Deposit adds funds to user account
// ⚠️ MOCK IMPLEMENTATION - No real payment gateway
func (ctrl *WalletController) Deposit(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseObjectID(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, err.Error())
		return
	}

	amountCents := utils.ToCents(req.Amount)

	user, transaction, err := ctrl.walletService.Deposit(c.Request.Context(), userID, amountCents)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, err.Error())
		return
	}

	utils.RespondSuccess(c, gin.H{
		"balance":        utils.FormatCents(user.BalanceCents),
		"balance_cents":  user.BalanceCents,
		"transaction_id": transaction.ID.Hex(),
	})
}

// GetBalance gets user's current balance
func (ctrl *WalletController) GetBalance(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseObjectID(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	balanceCents, err := ctrl.walletService.GetBalance(c.Request.Context(), userID)
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to fetch balance")
		return
	}

	utils.RespondSuccess(c, gin.H{
		"balance_cents":   balanceCents,
		"balance_display": utils.FormatCents(balanceCents),
	})
}
