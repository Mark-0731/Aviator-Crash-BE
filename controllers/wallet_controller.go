package controllers

import (
	"io"
	"net/http"

	"aviator-backend/middleware"
	"aviator-backend/services"
	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type WalletController struct {
	walletService      *services.WalletService
	nowPaymentsService *services.NOWPaymentsService
}

func NewWalletController() *WalletController {
	return &WalletController{
		walletService:      services.NewWalletService(),
		nowPaymentsService: services.NewNOWPaymentsService(),
	}
}

// Deposit creates a NOWPayments crypto deposit address
// POST /api/wallet/deposit
// Body: { "amount": 50.0, "sandbox_case": "success" }
func (ctrl *WalletController) Deposit(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseUUID(userIDStr)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("invalid_user_id")
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		SandboxCase string  `json:"sandbox_case"` // Sandbox: "success", "partially_paid", etc.
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, utils.ErrValidationError, "amount must be greater than 0")
		return
	}

	payment, err := ctrl.nowPaymentsService.CreateDeposit(c.Request.Context(), userID, req.Amount, req.SandboxCase)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("create_deposit_failed")
		utils.RespondWithError(c, utils.ErrInternalError, err.Error())
		return
	}

	utils.RespondSuccess(c, gin.H{
		"payment_id":  payment.PaymentID,
		"invoice_url": payment.PayAddress, // PayAddress field stores the invoice URL
		"status":      payment.Status,
		"amount_usd":  payment.PriceAmountUSD,
		"message":     "Click the invoice URL to complete payment. Your balance will be credited automatically.",
	})
}

// GetDepositStatus polls the status of an in-progress deposit
// GET /api/wallet/deposit/:payment_id
func (ctrl *WalletController) GetDepositStatus(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseUUID(userIDStr)
	if err != nil {
		utils.RespondWithError(c, utils.ErrBadRequest, "Invalid user ID")
		return
	}

	paymentID := c.Param("payment_id")
	if paymentID == "" {
		utils.RespondWithError(c, utils.ErrBadRequest, "payment_id is required")
		return
	}

	payment, err := ctrl.nowPaymentsService.GetPaymentStatus(c.Request.Context(), paymentID, userID)
	if err != nil {
		utils.RespondWithError(c, utils.ErrNotFound, "Payment not found")
		return
	}

	utils.RespondSuccess(c, gin.H{
		"payment_id":     payment.PaymentID,
		"status":         payment.Status,
		"pay_address":    payment.PayAddress,
		"pay_amount":     payment.PayAmount,
		"pay_currency":   payment.PayCurrency,
		"actually_paid":  payment.ActuallyPaid,
		"credited_cents": payment.CreditedCents,
		"amount_usd":     payment.PriceAmountUSD,
		"credited":       payment.Status == "finished",
	})
}

// NOWPaymentsWebhook receives IPN callbacks from NOWPayments
// POST /api/wallet/webhook/nowpayments (NO auth - verified by HMAC signature)
func (ctrl *WalletController) NOWPaymentsWebhook(c *gin.Context) {
	// Read raw body for signature verification
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("webhook_read_body_failed")
		c.Status(http.StatusBadRequest)
		return
	}

	// Verify HMAC-SHA512 signature
	sig := c.GetHeader("x-nowpayments-sig")
	ipnConfigured := ctrl.nowPaymentsService.IsIPNSecretConfigured()
	sigValid := sig != "" && ctrl.nowPaymentsService.VerifyIPNSignature(rawBody, sig)

	if !sigValid {
		if ipnConfigured {
			// Production: reject unsigned / tampered webhooks
			log.Warn().Msg("webhook_signature_verification_failed")
			c.Status(http.StatusUnauthorized)
			return
		}
		// Sandbox without IPN secret: allow but log so it's visible in logs
		log.Warn().Msg("webhook_signature_skipped_no_ipn_secret_configured")
	}

	// Parse the payload
	payload, err := ctrl.nowPaymentsService.ParseIPNPayload(rawBody)
	if err != nil {
		log.Error().Err(err).Msg("webhook_parse_payload_failed")
		c.Status(http.StatusBadRequest)
		return
	}

	if err := ctrl.nowPaymentsService.HandleWebhook(c.Request.Context(), *payload); err != nil {
		log.Error().Err(err).Msg("webhook_handle_failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

// GetBalance gets user's current balance
func (ctrl *WalletController) GetBalance(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := parseUUID(userIDStr)
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
