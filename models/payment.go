package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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

// CryptoPayment stores a NOWPayments deposit intent in MongoDB
type CryptoPayment struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         primitive.ObjectID `bson:"user_id" json:"user_id"`
	PaymentID      string             `bson:"payment_id" json:"payment_id"` // NOWPayments payment_id (invoice_id for invoice API)
	OrderID        string             `bson:"order_id" json:"order_id"`     // Unique order_id sent in webhook
	Status         PaymentStatus      `bson:"status" json:"status"`
	PriceAmountUSD float64            `bson:"price_amount_usd" json:"price_amount_usd"`             // USD amount user requested
	PayAmount      float64            `bson:"pay_amount" json:"pay_amount"`                         // Crypto amount to send
	PayCurrency    string             `bson:"pay_currency" json:"pay_currency"`                     // e.g. "usdttrc20"
	PayAddress     string             `bson:"pay_address" json:"pay_address"`                       // Crypto address or invoice URL
	ActuallyPaid   float64            `bson:"actually_paid" json:"actually_paid"`                   // Amount actually received
	CreditedCents  int64              `bson:"credited_cents" json:"credited_cents"`                 // Balance credited (0 until finished)
	SandboxCase    string             `bson:"sandbox_case,omitempty" json:"sandbox_case,omitempty"` // For sandbox testing
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}
