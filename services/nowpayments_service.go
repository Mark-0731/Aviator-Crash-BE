package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"aviator-backend/config"
	"aviator-backend/models"
	"aviator-backend/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

const nowPaymentsSandboxBase = "https://api-sandbox.nowpayments.io/v1"

// NOWPaymentsService handles the crypto deposit business logic.
// All DB access is delegated to PaymentRepository.
// All NOWPayments API calls are done via the internal http client.
type NOWPaymentsService struct {
	apiKey      string
	ipnSecret   string
	callbackURL string
	payCurrency string
	httpClient  *http.Client
	paymentRepo *repository.PaymentRepository
	walletSvc   *WalletService
}

func NewNOWPaymentsService() *NOWPaymentsService {
	return &NOWPaymentsService{
		apiKey:      config.AppConfig.NOWPaymentsAPIKey,
		ipnSecret:   config.AppConfig.NOWPaymentsIPNSecret,
		callbackURL: config.AppConfig.NOWPaymentsCallbackURL,
		payCurrency: config.AppConfig.NOWPaymentsPayCurrency,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		paymentRepo: repository.NewPaymentRepository(),
		walletSvc:   NewWalletService(),
	}
}

// ── NOWPayments API DTOs ────────────────────────────────────────────────────

type createInvoiceRequest struct {
	PriceAmount      float64 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayCurrency      string  `json:"pay_currency,omitempty"` // Default currency to show
	OrderID          string  `json:"order_id"`
	OrderDescription string  `json:"order_description"`
	IPNCallbackURL   string  `json:"ipn_callback_url"`
	SuccessURL       string  `json:"success_url,omitempty"`
	CancelURL        string  `json:"cancel_url,omitempty"`
}

type createInvoiceResponse struct {
	ID          string `json:"id"`
	InvoiceURL  string `json:"invoice_url"`
	OrderID     string `json:"order_id"`
	PriceAmount string `json:"price_amount"`
	Status      string `json:"payment_status"`
}

// IPNPayload is the NOWPayments IPN webhook body
type IPNPayload struct {
	PaymentID    interface{} `json:"payment_id"` // Can be int or string depending on NOWPayments version
	Status       string      `json:"payment_status"`
	ActuallyPaid float64     `json:"actually_paid"`
	PayAmount    float64     `json:"pay_amount"`
	OrderID      string      `json:"order_id"`
}

// ── Public Methods ──────────────────────────────────────────────────────────

// CreateDeposit calls NOWPayments to create an invoice with hosted payment page.
func (s *NOWPaymentsService) CreateDeposit(ctx context.Context, userID uuid.UUID, amountUSD float64, sandboxCase string) (*models.CryptoPayment, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("NOWPayments API key not configured — set NOWPAYMENTS_API_KEY in .env")
	}

	orderID := fmt.Sprintf("%s:%d", userID.String(), time.Now().UnixMilli())

	reqBody := createInvoiceRequest{
		PriceAmount:      amountUSD,
		PriceCurrency:    "usd",
		PayCurrency:      s.payCurrency, // Default to USDT TRC20
		OrderID:          orderID,
		OrderDescription: fmt.Sprintf("Aviator Game Deposit - $%.2f", amountUSD),
		IPNCallbackURL:   s.callbackURL,
		SuccessURL:       "http://localhost:3000/game?deposit=success",
		CancelURL:        "http://localhost:3000/game?deposit=cancelled",
	}

	rawResp, err := s.post("/invoice", reqBody)
	if err != nil {
		return nil, fmt.Errorf("nowpayments create_invoice: %w", err)
	}

	var nwResp createInvoiceResponse
	if err := json.Unmarshal(rawResp, &nwResp); err != nil {
		return nil, fmt.Errorf("nowpayments parse invoice response: %w", err)
	}

	payment := &models.CryptoPayment{
		ID:             uuid.New(),
		UserID:         userID,
		PaymentID:      nwResp.ID,
		OrderID:        orderID, // Store order_id for webhook lookup
		Status:         models.PaymentStatus(nwResp.Status),
		PriceAmountUSD: amountUSD,
		PayAddress:     nwResp.InvoiceURL, // Store invoice URL in PayAddress field
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("persist payment: %w", err)
	}

	return payment, nil
}

// GetPaymentStatus fetches payment from DB (for frontend polling).
func (s *NOWPaymentsService) GetPaymentStatus(ctx context.Context, paymentID string, userID uuid.UUID) (*models.CryptoPayment, error) {
	payment, err := s.paymentRepo.FindByPaymentIDAndUser(ctx, paymentID, userID)
	if err != nil {
		// FindByPaymentIDAndUser wraps pgx.ErrNoRows as errors.New("payment not found")
		return nil, fmt.Errorf("payment not found")
	}
	return payment, nil
}

// VerifyIPNSignature validates the x-nowpayments-sig header.
// NOWPayments signs the alphabetically-sorted JSON body with HMAC-SHA512.
func (s *NOWPaymentsService) VerifyIPNSignature(body []byte, receivedSig string) bool {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	// Sort keys and re-encode
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make(map[string]interface{}, len(payload))
	for _, k := range keys {
		ordered[k] = payload[k]
	}
	sortedJSON, err := json.Marshal(ordered)
	if err != nil {
		return false
	}

	mac := hmac.New(sha512.New, []byte(s.ipnSecret))
	mac.Write(sortedJSON)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(receivedSig))
}

// IsIPNSecretConfigured returns true if an IPN secret is set.
func (s *NOWPaymentsService) IsIPNSecretConfigured() bool {
	return s.ipnSecret != ""
}

// ParseIPNPayload unmarshals a raw webhook body into IPNPayload.
func (s *NOWPaymentsService) ParseIPNPayload(body []byte) (*IPNPayload, error) {
	var p IPNPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("parse ipn payload: %w", err)
	}
	return &p, nil
}

// HandleWebhook processes a verified IPN and credits the user balance — idempotent.
// Uses order_id for lookup since webhook sends payment_id (different from invoice_id).
func (s *NOWPaymentsService) HandleWebhook(ctx context.Context, payload IPNPayload) error {
	orderID := payload.OrderID
	newStatus := models.PaymentStatus(payload.Status)

	// Idempotency: skip if already finished
	alreadyDone, err := s.paymentRepo.IsAlreadyFinishedByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("check payment status: %w", err)
	}
	if alreadyDone {
		return nil
	}

	// Fetch payment by order_id (not payment_id, since webhook sends different ID)
	payment, err := s.paymentRepo.FindByOrderID(ctx, orderID)
	if err == pgx.ErrNoRows {
		return nil // Unknown payment — ignore gracefully
	}
	if err != nil {
		return fmt.Errorf("find payment: %w", err)
	}

	// Credit balance only when fully finished
	var creditedCents int64
	if newStatus == models.PaymentStatusFinished {
		amountUSD := payload.ActuallyPaid
		if amountUSD <= 0 {
			amountUSD = payment.PriceAmountUSD
		}
		creditedCents = int64(amountUSD * 100) // 1 USD = $1.00 in game cents

		if _, _, err := s.walletSvc.Deposit(ctx, payment.UserID, creditedCents); err != nil {
			log.Error().Err(err).Str("order_id", orderID).Msg("credit_balance_failed")
			return fmt.Errorf("credit balance: %w", err)
		}

		log.Info().
			Str("order_id", orderID).
			Str("user_id", payment.UserID.String()).
			Int64("credited_cents", creditedCents).
			Msg("crypto_deposit_credited")
	}

	// Persist the new status via repository using order_id
	return s.paymentRepo.UpdateStatusByOrderID(ctx, orderID, newStatus, payload.ActuallyPaid, creditedCents)
}

// ── Private HTTP helper ─────────────────────────────────────────────────────

func (s *NOWPaymentsService) post(path string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := nowPaymentsSandboxBase + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response", string(respBytes)).
			Str("path", path).
			Msg("nowpayments_api_error")
		return nil, fmt.Errorf("nowpayments API %s returned %d: %s", path, resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}
