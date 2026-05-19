package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus tracks NOWPayments payment lifecycle
type PaymentStatus string

const (
	PaymentStatusWaiting       PaymentStatus = "waiting"
	PaymentStatusConfirming    PaymentStatus = "confirming"
	PaymentStatusConfirmed     PaymentStatus = "confirmed"
	PaymentStatusSending       PaymentStatus = "sending"
	PaymentStatusPartiallyPaid PaymentStatus = "partially_paid"
	PaymentStatusFinished      PaymentStatus = "finished"
	PaymentStatusFailed        PaymentStatus = "failed"
	PaymentStatusRefunded      PaymentStatus = "refunded"
	PaymentStatusExpired       PaymentStatus = "expired"
)

// CryptoPayment stores a NOWPayments deposit intent
type CryptoPayment struct {
	ID             uuid.UUID     `json:"id"`
	UserID         uuid.UUID     `json:"user_id"`
	PaymentID      string        `json:"payment_id"` // NOWPayments payment_id (invoice_id for invoice API)
	OrderID        string        `json:"order_id"`   // Unique order_id sent in webhook
	Status         PaymentStatus `json:"status"`
	PriceAmountUSD float64       `json:"price_amount_usd"`       // USD amount user requested
	PayAmount      float64       `json:"pay_amount"`             // Crypto amount to send
	PayCurrency    string        `json:"pay_currency"`           // e.g. "usdttrc20"
	PayAddress     string        `json:"pay_address"`            // Crypto address or invoice URL
	ActuallyPaid   float64       `json:"actually_paid"`          // Amount actually received
	CreditedCents  int64         `json:"credited_cents"`         // Balance credited (0 until finished)
	SandboxCase    string        `json:"sandbox_case,omitempty"` // For sandbox testing
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
