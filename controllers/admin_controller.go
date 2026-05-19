package controllers

import (
	"aviator-backend/services"
	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

type AdminController struct {
	adminService *services.AdminService
}

func NewAdminController() *AdminController {
	return &AdminController{
		adminService: services.NewAdminService(),
	}
}

// GetUsers gets all users with pagination and search
func (ctrl *AdminController) GetUsers(c *gin.Context) {
	page, limit := parsePagination(c, 20)
	search := c.DefaultQuery("search", "")

	users, total, err := ctrl.adminService.GetUsers(c.Request.Context(), page, limit, search)
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to fetch users")
		return
	}

	utils.RespondSuccess(c, paginationResponse(users, total, page, limit))
}

// BanUser bans a user
func (ctrl *AdminController) BanUser(c *gin.Context) {
	userID, err := parseUUID(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, err.Error())
		return
	}

	if err := ctrl.adminService.BanUser(c.Request.Context(), userID, req.Reason); err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to ban user")
		return
	}

	utils.RespondSuccess(c, gin.H{"message": "User banned successfully"})
}

// UnbanUser unbans a user
func (ctrl *AdminController) UnbanUser(c *gin.Context) {
	userID, err := parseUUID(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	if err := ctrl.adminService.UnbanUser(c.Request.Context(), userID); err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to unban user")
		return
	}

	utils.RespondSuccess(c, gin.H{"message": "User unbanned successfully"})
}

// AdjustBalance adjusts a user's balance
func (ctrl *AdminController) AdjustBalance(c *gin.Context) {
	userID, err := parseUUID(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		AmountCents int64  `json:"amount_cents" binding:"required"`
		Reason      string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, err.Error())
		return
	}

	user, err := ctrl.adminService.AdjustBalance(c.Request.Context(), userID, req.AmountCents, req.Reason)
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, err.Error())
		return
	}

	utils.RespondSuccess(c, gin.H{
		"message": "Balance adjusted successfully",
		"user":    user.ToResponse(),
	})
}

// GetRounds gets all rounds with pagination
func (ctrl *AdminController) GetRounds(c *gin.Context) {
	page, limit := parsePagination(c, 20)

	rounds, total, err := ctrl.adminService.GetAllRounds(c.Request.Context(), page, limit)
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to fetch rounds")
		return
	}

	utils.RespondSuccess(c, paginationResponse(rounds, total, page, limit))
}

// GetStats gets aggregate statistics
func (ctrl *AdminController) GetStats(c *gin.Context) {
	stats, err := ctrl.adminService.GetStats(c.Request.Context())
	if err != nil {
		utils.RespondWithError(c, utils.ErrInternalError, "Failed to fetch stats")
		return
	}

	utils.RespondSuccess(c, stats)
}
