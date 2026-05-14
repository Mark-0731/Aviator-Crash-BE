package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BetStatus represents the state of a bet
type BetStatus string

const (
	BetStatusPending  BetStatus = "pending"
	BetStatusWon      BetStatus = "won"
	BetStatusLost     BetStatus = "lost"
	BetStatusRefunded BetStatus = "refunded"
)

// Bet represents a player's wager in a round
type Bet struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID                primitive.ObjectID `bson:"user_id" json:"user_id"`
	RoundID               string             `bson:"round_id" json:"round_id"`
	AmountCents           int64              `bson:"amount_cents" json:"amount_cents"`
	CashoutMultiplierX100 *int64             `bson:"cashout_multiplier_x100,omitempty" json:"cashout_multiplier_x100,omitempty"` // nullable
	ProfitCents           int64              `bson:"profit_cents" json:"profit_cents"`
	Status                BetStatus          `bson:"status" json:"status"`
	PlacedAt              time.Time          `bson:"placed_at" json:"placed_at"`
	CashedOutAt           *time.Time         `bson:"cashed_out_at,omitempty" json:"cashed_out_at,omitempty"` // nullable
}

// BetResponse is the API representation of a bet
type BetResponse struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Username          string     `json:"username,omitempty"`
	RoundID           string     `json:"round_id"`
	Amount            string     `json:"amount"` // formatted as "$50.00"
	AmountCents       int64      `json:"amount_cents"`
	CashoutMultiplier *float64   `json:"cashout_multiplier,omitempty"` // e.g., 2.45
	Profit            string     `json:"profit"`                       // formatted as "$55.00"
	ProfitCents       int64      `json:"profit_cents"`
	Status            BetStatus  `json:"status"`
	PlacedAt          time.Time  `json:"placed_at"`
	CashedOutAt       *time.Time `json:"cashed_out_at,omitempty"`
}

// ToResponse converts Bet to BetResponse
func (b *Bet) ToResponse() BetResponse {
	resp := BetResponse{
		ID:          b.ID.Hex(),
		UserID:      b.UserID.Hex(),
		RoundID:     b.RoundID,
		Amount:      FormatCents(b.AmountCents),
		AmountCents: b.AmountCents,
		Profit:      FormatCents(b.ProfitCents),
		ProfitCents: b.ProfitCents,
		Status:      b.Status,
		PlacedAt:    b.PlacedAt,
		CashedOutAt: b.CashedOutAt,
	}

	if b.CashoutMultiplierX100 != nil {
		multiplier := convertX100ToFloat(*b.CashoutMultiplierX100)
		resp.CashoutMultiplier = &multiplier
	}

	return resp
}
