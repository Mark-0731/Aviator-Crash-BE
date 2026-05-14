package repository

import (
	"context"
	"time"

	"aviator-backend/database"
	"aviator-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthRepository struct {
	refreshTokens *BaseRepository
	wsTickets     *BaseRepository
}

func NewAuthRepository() *AuthRepository {
	return &AuthRepository{
		refreshTokens: newBase(database.DB.Collection("refresh_tokens")),
		wsTickets:     newBase(database.DB.Collection("ws_tickets")),
	}
}

// Refresh Token Operations

func (r *AuthRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	token.CreatedAt = time.Now()
	id, err := r.refreshTokens.InsertOne(ctx, token)
	if err != nil {
		return err
	}
	token.ID = id
	return nil
}

func (r *AuthRepository) FindRefreshToken(ctx context.Context, tokenString string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.refreshTokens.FindOne(ctx, bson.D{{Key: "token", Value: tokenString}}, &token)
	return &token, err
}

func (r *AuthRepository) DeleteRefreshToken(ctx context.Context, tokenString string) error {
	return r.refreshTokens.DeleteOne(ctx, bson.D{{Key: "token", Value: tokenString}})
}

func (r *AuthRepository) DeleteUserRefreshTokens(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.refreshTokens.DeleteMany(ctx, bson.D{{Key: "user_id", Value: userID}})
	return err
}

func (r *AuthRepository) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	return r.refreshTokens.DeleteExpired(ctx)
}

// WebSocket Ticket Operations

func (r *AuthRepository) CreateWSTicket(ctx context.Context, ticket *models.WSTicket) error {
	ticket.CreatedAt = time.Now()
	id, err := r.wsTickets.InsertOne(ctx, ticket)
	if err != nil {
		return err
	}
	ticket.ID = id
	return nil
}

func (r *AuthRepository) FindWSTicket(ctx context.Context, ticketString string) (*models.WSTicket, error) {
	var ticket models.WSTicket
	err := r.wsTickets.FindOne(ctx, bson.D{{Key: "ticket", Value: ticketString}}, &ticket)
	return &ticket, err
}

func (r *AuthRepository) MarkWSTicketUsed(ctx context.Context, ticketString string) error {
	return r.wsTickets.UpdateOne(ctx,
		bson.D{{Key: "ticket", Value: ticketString}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "used", Value: true}}}},
	)
}

func (r *AuthRepository) DeleteExpiredWSTickets(ctx context.Context) (int64, error) {
	return r.wsTickets.DeleteExpired(ctx)
}
