package repository

import (
	"context"
	"errors"
	"time"

	"aviator-backend/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AuthRepository struct {
	*BaseRepository
}

func NewAuthRepository() *AuthRepository {
	return &AuthRepository{
		BaseRepository: newBase("refresh_tokens"),
	}
}

// Refresh Token Operations

func (r *AuthRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	token.CreatedAt = time.Now()

	query := `
		INSERT INTO refresh_tokens (token, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := r.getTx(ctx).QueryRow(ctx, query,
		token.Token, token.UserID, token.ExpiresAt, token.CreatedAt,
	).Scan(&token.ID)

	return err
}

func (r *AuthRepository) FindRefreshToken(ctx context.Context, tokenString string) (*models.RefreshToken, error) {
	query := `SELECT id, token, user_id, expires_at, created_at FROM refresh_tokens WHERE token = $1`

	var token models.RefreshToken
	err := r.getTx(ctx).QueryRow(ctx, query, tokenString).Scan(
		&token.ID, &token.Token, &token.UserID, &token.ExpiresAt, &token.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("refresh token not found")
		}
		return nil, err
	}

	return &token, nil
}

func (r *AuthRepository) DeleteRefreshToken(ctx context.Context, tokenString string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := r.getTx(ctx).Exec(ctx, query, tokenString)
	return err
}

func (r *AuthRepository) DeleteUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.getTx(ctx).Exec(ctx, query, userID)
	return err
}

func (r *AuthRepository) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`
	result, err := r.getTx(ctx).Exec(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// WebSocket Ticket Operations

func (r *AuthRepository) CreateWSTicket(ctx context.Context, ticket *models.WSTicket) error {
	ticket.CreatedAt = time.Now()

	query := `
		INSERT INTO ws_tickets (ticket, user_id, used, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	err := r.getTx(ctx).QueryRow(ctx, query,
		ticket.Ticket, ticket.UserID, ticket.Used, ticket.ExpiresAt, ticket.CreatedAt,
	).Scan(&ticket.ID)

	return err
}

func (r *AuthRepository) FindWSTicket(ctx context.Context, ticketString string) (*models.WSTicket, error) {
	query := `SELECT id, ticket, user_id, used, expires_at, created_at FROM ws_tickets WHERE ticket = $1`

	var ticket models.WSTicket
	err := r.getTx(ctx).QueryRow(ctx, query, ticketString).Scan(
		&ticket.ID, &ticket.Ticket, &ticket.UserID, &ticket.Used, &ticket.ExpiresAt, &ticket.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("ws ticket not found")
		}
		return nil, err
	}

	return &ticket, nil
}

func (r *AuthRepository) MarkWSTicketUsed(ctx context.Context, ticketString string) error {
	query := `UPDATE ws_tickets SET used = true WHERE ticket = $1`
	_, err := r.getTx(ctx).Exec(ctx, query, ticketString)
	return err
}

func (r *AuthRepository) DeleteExpiredWSTickets(ctx context.Context) (int64, error) {
	query := `DELETE FROM ws_tickets WHERE expires_at < $1`
	result, err := r.getTx(ctx).Exec(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
