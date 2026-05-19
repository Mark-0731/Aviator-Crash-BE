package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a player account
type User struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Password       string    `json:"-"` // NEVER returned in any response
	BalanceCents   int64     `json:"balance_cents"`
	ClientSeed     string    `json:"client_seed"` // Player-provided seed for provably fair
	IsAdmin        bool      `json:"is_admin"`
	IsBanned       bool      `json:"is_banned"`
	BanReason      string    `json:"ban_reason,omitempty"`
	RegistrationIP string    `json:"-"`
	LastLoginIP    string    `json:"-"`
	LastLoginAt    time.Time `json:"last_login_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserResponse is the safe user representation for API responses
type UserResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	BalanceCents int64     `json:"balance_cents"`
	Balance      string    `json:"balance"`     // formatted as "$1000.00"
	ClientSeed   string    `json:"client_seed"` // Player's current client seed
	IsAdmin      bool      `json:"is_admin"`
	IsBanned     bool      `json:"is_banned"`
	BanReason    string    `json:"ban_reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ToResponse converts User to UserResponse (safe for API)
func (u *User) ToResponse() UserResponse {
	clientSeed := u.ClientSeed
	if clientSeed == "" {
		clientSeed = "default" // Not yet set by player
	}
	return UserResponse{
		ID:           u.ID.String(),
		Username:     u.Username,
		Email:        u.Email,
		BalanceCents: u.BalanceCents,
		Balance:      FormatCents(u.BalanceCents),
		ClientSeed:   clientSeed,
		IsAdmin:      u.IsAdmin,
		IsBanned:     u.IsBanned,
		BanReason:    u.BanReason,
		CreatedAt:    u.CreatedAt,
	}
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// WSTicket represents a one-time WebSocket authentication ticket
type WSTicket struct {
	ID        uuid.UUID `json:"id"`
	Ticket    string    `json:"ticket"`
	UserID    uuid.UUID `json:"user_id"`
	Used      bool      `json:"used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
