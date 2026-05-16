package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RoundStatus represents the state of a game round
type RoundStatus string

const (
	RoundStatusWaiting    RoundStatus = "waiting"
	RoundStatusRunning    RoundStatus = "running"
	RoundStatusCrashed    RoundStatus = "crashed"
	RoundStatusRecovering RoundStatus = "recovering"
)

// Round represents a single game round
type Round struct {
	ID                        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RoundID                   string             `bson:"round_id" json:"round_id"`                 // UUID v4
	CrashPointX100            int64              `bson:"crash_point_x100" json:"crash_point_x100"` // NEVER sent before crash
	ServerSeed                string             `bson:"server_seed" json:"server_seed"`           // NEVER sent before crash
	ServerSeedHash            string             `bson:"server_seed_hash" json:"server_seed_hash"` // SHA256 commitment
	ClientSeed                string             `bson:"client_seed" json:"client_seed"`
	Nonce                     int64              `bson:"nonce" json:"nonce"`
	Hash                      string             `bson:"hash" json:"hash"` // HMAC-SHA256 hex
	Status                    RoundStatus        `bson:"status" json:"status"`
	TotalWageredCents         int64              `bson:"total_wagered_cents" json:"total_wagered_cents"`
	TotalPayoutCents          int64              `bson:"total_payout_cents" json:"total_payout_cents"`
	PlayerCount               int                `bson:"player_count" json:"player_count"`
	ConsecutiveInstantCrashes int                `bson:"consecutive_instant_crashes" json:"consecutive_instant_crashes"`
	StartedAt                 time.Time          `bson:"started_at" json:"started_at"`
	CrashedAt                 *time.Time         `bson:"crashed_at,omitempty" json:"crashed_at,omitempty"`
	TransitionTime            time.Time          `bson:"transition_time" json:"transition_time"` // For bet grace window
}

// RoundResponse is the API representation of a round
type RoundResponse struct {
	ID                string      `json:"id"`
	RoundID           string      `json:"round_id"`
	CrashPoint        *float64    `json:"crash_point,omitempty"` // Only included after crash
	ServerSeed        *string     `json:"server_seed,omitempty"` // Only included after crash
	ServerSeedHash    string      `json:"server_seed_hash"`
	ClientSeed        string      `json:"client_seed"`
	Nonce             int64       `json:"nonce"`
	Hash              string      `json:"hash"`
	Status            RoundStatus `json:"status"`
	TotalWagered      string      `json:"total_wagered"`
	TotalWageredCents int64       `json:"total_wagered_cents"`
	TotalPayout       string      `json:"total_payout"`
	TotalPayoutCents  int64       `json:"total_payout_cents"`
	HouseProfit       string      `json:"house_profit"`
	HouseProfitCents  int64       `json:"house_profit_cents"`
	PlayerCount       int         `json:"player_count"`
	StartedAt         time.Time   `json:"started_at"`
	CrashedAt         *time.Time  `json:"crashed_at,omitempty"`
}

// ToResponse converts Round to RoundResponse
// CRITICAL: Only includes crash_point and server_seed if round is crashed
func (r *Round) ToResponse() RoundResponse {
	houseProfitCents := calculateHouseProfit(r.TotalWageredCents, r.TotalPayoutCents)

	resp := RoundResponse{
		ID:                r.ID.Hex(),
		RoundID:           r.RoundID,
		ServerSeedHash:    r.ServerSeedHash,
		ClientSeed:        r.ClientSeed,
		Nonce:             r.Nonce,
		Hash:              r.Hash,
		Status:            r.Status,
		TotalWagered:      FormatCents(r.TotalWageredCents),
		TotalWageredCents: r.TotalWageredCents,
		TotalPayout:       FormatCents(r.TotalPayoutCents),
		TotalPayoutCents:  r.TotalPayoutCents,
		HouseProfit:       FormatCents(houseProfitCents),
		HouseProfitCents:  houseProfitCents,
		PlayerCount:       r.PlayerCount,
		StartedAt:         r.StartedAt,
		CrashedAt:         r.CrashedAt,
	}

	// SECURITY: Only reveal crash_point and server_seed after round has crashed
	if r.Status == RoundStatusCrashed {
		crashPoint := convertX100ToFloat(r.CrashPointX100)
		resp.CrashPoint = &crashPoint
		resp.ServerSeed = &r.ServerSeed
	}

	return resp
}
