package services

import (
	"context"
	"errors"

	"aviator-backend/models"
	"aviator-backend/repository"

	"github.com/google/uuid"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

type AdminService struct {
	userRepo        *repository.UserRepository
	roundRepo       *repository.RoundRepository
	transactionRepo *repository.TransactionRepository
}

func NewAdminService() *AdminService {
	return &AdminService{
		userRepo:        repository.NewUserRepository(),
		roundRepo:       repository.NewRoundRepository(),
		transactionRepo: repository.NewTransactionRepository(),
	}
}

// GetUsers gets all users with pagination and search
func (s *AdminService) GetUsers(ctx context.Context, page, limit int64, search string) ([]models.UserResponse, int64, error) {
	users, total, err := s.userRepo.FindAll(ctx, page, limit, search)
	if err != nil {
		return nil, 0, err
	}
	return convertToUserResponses(users), total, nil
}

// BanUser bans a user with a reason
func (s *AdminService) BanUser(ctx context.Context, userID uuid.UUID, reason string) error {
	if reason == "" {
		return errors.New("ban reason is required")
	}
	return s.userRepo.BanUser(ctx, userID, reason)
}

// UnbanUser unbans a user
func (s *AdminService) UnbanUser(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.UnbanUser(ctx, userID)
}

// AdjustBalance adjusts a user's balance (admin operation)
func (s *AdminService) AdjustBalance(ctx context.Context, userID uuid.UUID, amountCents int64, reason string) (*models.User, error) {
	if reason == "" {
		return nil, errors.New("reason is required")
	}

	// AdjustBalance uses UpdateBalance which returns the user post-update (RETURNING clause)
	updatedUser, err := s.userRepo.AdjustBalance(ctx, userID, amountCents)
	if err != nil {
		return nil, err
	}

	// Derive balance_before from post-update value — no separate FindByID needed
	balanceBefore := updatedUser.BalanceCents - amountCents

	transaction := &models.Transaction{
		UserID:             userID,
		Type:               models.TransactionTypeAdjustment,
		AmountCents:        abs(amountCents),
		BalanceBeforeCents: balanceBefore,
		BalanceAfterCents:  updatedUser.BalanceCents,
		Reason:             reason,
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	return updatedUser, nil
}

// GetAllRounds gets all rounds with pagination (admin view)
func (s *AdminService) GetAllRounds(ctx context.Context, page, limit int64) ([]models.RoundResponse, int64, error) {
	rounds, total, err := s.roundRepo.FindAll(ctx, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToRoundResponses(rounds), total, nil
}

// GetStats gets aggregate statistics
func (s *AdminService) GetStats(ctx context.Context) (map[string]any, error) {
	roundStats, err := s.roundRepo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	totalUsers, err := s.userRepo.CountAll(ctx)
	if err != nil {
		totalUsers = 0
	}

	return map[string]any{
		"total_users":   totalUsers,
		"total_rounds":  roundStats["total_rounds"],
		"total_wagered": roundStats["total_wagered"],
		"total_payout":  roundStats["total_payout"],
		"house_profit":  roundStats["house_profit"],
	}, nil
}
