package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a player account
type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username       string             `bson:"username" json:"username"`
	Email          string             `bson:"email" json:"email"`
	Password       string             `bson:"password" json:"-"` // NEVER returned in any response
	BalanceCents   int64              `bson:"balance_cents" json:"balance_cents"`
	ClientSeed     string             `bson:"client_seed" json:"client_seed"` // Player-provided seed for provably fair
	IsAdmin        bool               `bson:"is_admin" json:"is_admin"`
	IsBanned       bool               `bson:"is_banned" json:"is_banned"`
	BanReason      string             `bson:"ban_reason,omitempty" json:"ban_reason,omitempty"`
	RegistrationIP string             `bson:"registration_ip" json:"-"`
	LastLoginIP    string             `bson:"last_login_ip" json:"-"`
	LastLoginAt    time.Time          `bson:"last_login_at" json:"last_login_at"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// UserResponse is the safe user representation for API responses
type UserResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	BalanceCents int64     `json:"balance_cents"`
	Balance      string    `json:"balance"` // formatted as "$1000.00"
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
		ID:           u.ID.Hex(),
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
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Token     string             `bson:"token" json:"token"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// WSTicket represents a one-time WebSocket authentication ticket
type WSTicket struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Ticket    string             `bson:"ticket" json:"ticket"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Used      bool               `bson:"used" json:"used"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
