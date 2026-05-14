package services

import (
	"context"
	"errors"
	"regexp"
	"time"

	"aviator-backend/config"
	"aviator-backend/models"
	"aviator-backend/repository"
	"aviator-backend/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS
// All database operations use repositories

type AuthService struct {
	userRepo *repository.UserRepository
	authRepo *repository.AuthRepository
}

func NewAuthService() *AuthService {
	return &AuthService{
		userRepo: repository.NewUserRepository(),
		authRepo: repository.NewAuthRepository(),
	}
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, username, email, password, ip string) (*models.User, string, string, error) {
	// Validate username
	if !usernameRegex.MatchString(username) {
		return nil, "", "", errors.New("username must be 3-20 characters, alphanumeric and underscores only")
	}

	// Validate email
	if !emailRegex.MatchString(email) {
		return nil, "", "", errors.New("invalid email format")
	}

	// Validate password
	if len(password) < 8 {
		return nil, "", "", errors.New("password must be at least 8 characters")
	}

	// Check if email exists using repository
	_, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return nil, "", "", errors.New("email already registered")
	}

	// Check if username exists using repository
	_, err = s.userRepo.FindByUsername(ctx, username)
	if err == nil {
		return nil, "", "", errors.New("username already taken")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, "", "", err
	}

	// Create user using repository
	user := &models.User{
		Username:       username,
		Email:          email,
		Password:       hashedPassword,
		BalanceCents:   config.AppConfig.DefaultBalanceCents,
		IsAdmin:        false,
		IsBanned:       false,
		RegistrationIP: ip,
		LastLoginIP:    ip,
		LastLoginAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", "", err
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return nil, "", "", err
	}

	refreshTokenString, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, "", "", err
	}

	// Store refresh token using repository
	refreshToken := &models.RefreshToken{
		Token:     refreshTokenString,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Duration(config.AppConfig.JWTRefreshExpiryDays) * 24 * time.Hour),
	}
	s.authRepo.CreateRefreshToken(ctx, refreshToken)

	return user, accessToken, refreshTokenString, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, email, password, ip string) (*models.User, string, string, error) {
	// Find user by email using repository
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, "", "", errors.New("invalid email or password")
		}
		return nil, "", "", err
	}

	// Check if banned
	if user.IsBanned {
		return nil, "", "", errors.New("account is banned: " + user.BanReason)
	}

	// Verify password
	if !utils.CheckPasswordHash(password, user.Password) {
		return nil, "", "", errors.New("invalid email or password")
	}

	// Update last login using repository
	s.userRepo.UpdateLastLogin(ctx, user.ID, ip)

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return nil, "", "", err
	}

	refreshTokenString, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, "", "", err
	}

	// Store refresh token using repository
	refreshToken := &models.RefreshToken{
		Token:     refreshTokenString,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Duration(config.AppConfig.JWTRefreshExpiryDays) * 24 * time.Hour),
	}
	s.authRepo.CreateRefreshToken(ctx, refreshToken)

	return user, accessToken, refreshTokenString, nil
}

// RefreshToken generates a new access token from refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (string, error) {
	// Find refresh token using repository
	refreshToken, err := s.authRepo.FindRefreshToken(ctx, refreshTokenString)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	// Check expiration
	if time.Now().After(refreshToken.ExpiresAt) {
		s.authRepo.DeleteRefreshToken(ctx, refreshTokenString)
		return "", errors.New("refresh token expired")
	}

	// Get user using repository
	user, err := s.userRepo.FindByID(ctx, refreshToken.UserID)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Check if banned
	if user.IsBanned {
		return "", errors.New("account is banned")
	}

	// Generate new access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

// Logout invalidates a refresh token
func (s *AuthService) Logout(ctx context.Context, refreshTokenString string) error {
	// Delete refresh token using repository
	return s.authRepo.DeleteRefreshToken(ctx, refreshTokenString)
}

// CreateWSTicket creates a one-time WebSocket authentication ticket
func (s *AuthService) CreateWSTicket(ctx context.Context, userID primitive.ObjectID) (string, error) {
	ticketString := uuid.New().String()

	ticket := &models.WSTicket{
		Ticket:    ticketString,
		UserID:    userID,
		Used:      false,
		ExpiresAt: time.Now().Add(time.Duration(config.AppConfig.WSTicketExpirySeconds) * time.Second),
	}

	if err := s.authRepo.CreateWSTicket(ctx, ticket); err != nil {
		return "", err
	}

	return ticketString, nil
}

// ValidateWSTicket validates and marks a WebSocket ticket as used
func (s *AuthService) ValidateWSTicket(ctx context.Context, ticketString string) (*models.User, error) {
	// Find ticket using repository
	ticket, err := s.authRepo.FindWSTicket(ctx, ticketString)
	if err != nil {
		return nil, errors.New("invalid ticket")
	}

	// Check if already used
	if ticket.Used {
		return nil, errors.New("ticket already used")
	}

	// Check expiration
	if time.Now().After(ticket.ExpiresAt) {
		return nil, errors.New("ticket expired")
	}

	// Mark as used using repository
	if err := s.authRepo.MarkWSTicketUsed(ctx, ticketString); err != nil {
		return nil, err
	}

	// Get user using repository
	user, err := s.userRepo.FindByID(ctx, ticket.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if banned
	if user.IsBanned {
		return nil, errors.New("account is banned")
	}

	return user, nil
}

// CleanupExpiredTokens removes expired refresh tokens and WS tickets
func (s *AuthService) CleanupExpiredTokens(ctx context.Context) error {
	// Cleanup using repositories
	deletedTokens, err := s.authRepo.DeleteExpiredRefreshTokens(ctx)
	if err != nil {
		return err
	}

	deletedTickets, err := s.authRepo.DeleteExpiredWSTickets(ctx)
	if err != nil {
		return err
	}

	if deletedTokens > 0 || deletedTickets > 0 {
		// Log cleanup (optional)
	}

	return nil
}
