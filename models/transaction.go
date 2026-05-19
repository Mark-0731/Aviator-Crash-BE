package models

import (
	"time"

	"github.com/google/uuid"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeBet        TransactionType = "bet"
	TransactionTypeWin        TransactionType = "win"
	TransactionTypeRefund     TransactionType = "refund"
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
	TransactionTypeAdjustment TransactionType = "adjustment" // Admin balance adjustment
)

// Transaction represents a balance change event
type Transaction struct {
	ID                 uuid.UUID       `json:"id"`
	UserID             uuid.UUID       `json:"user_id"`
	Type               TransactionType `json:"type"`
	AmountCents        int64           `json:"amount_cents"`       // Always positive
	RoundID            *string         `json:"round_id,omitempty"` // Only for bet/win/refund
	BalanceBeforeCents int64           `json:"balance_before_cents"`
	BalanceAfterCents  int64           `json:"balance_after_cents"`
	Reason             string          `json:"reason,omitempty"` // For admin adjustments
	CreatedAt          time.Time       `json:"created_at"`
}

// TransactionResponse is the API representation of a transaction
type TransactionResponse struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	Type          TransactionType `json:"type"`
	Amount        string          `json:"amount"`
	AmountCents   int64           `json:"amount_cents"`
	RoundID       *string         `json:"round_id,omitempty"`
	BalanceBefore string          `json:"balance_before"`
	BalanceAfter  string          `json:"balance_after"`
	Reason        string          `json:"reason,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ToResponse converts Transaction to TransactionResponse
func (t *Transaction) ToResponse() TransactionResponse {
	return TransactionResponse{
		ID:            t.ID.String(),
		UserID:        t.UserID.String(),
		Type:          t.Type,
		Amount:        FormatCents(t.AmountCents),
		AmountCents:   t.AmountCents,
		RoundID:       t.RoundID,
		BalanceBefore: FormatCents(t.BalanceBeforeCents),
		BalanceAfter:  FormatCents(t.BalanceAfterCents),
		Reason:        t.Reason,
		CreatedAt:     t.CreatedAt,
	}
}
