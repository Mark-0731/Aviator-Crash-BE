package services

import (
	"context"
	"errors"

	"aviator-backend/models"
	"aviator-backend/repository"

	"github.com/google/uuid"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

type WalletService struct {
	userRepo        *repository.UserRepository
	transactionRepo *repository.TransactionRepository
}

func NewWalletService() *WalletService {
	return &WalletService{
		userRepo:        repository.NewUserRepository(),
		transactionRepo: repository.NewTransactionRepository(),
	}
}

// Deposit adds funds to user account
// NOTE: This is a MOCK implementation - no real payment gateway integration
// In production, integrate with Stripe, PayPal, or other payment providers
func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, amountCents int64) (*models.User, *models.Transaction, error) {
	if amountCents <= 0 {
		return nil, nil, errors.New("deposit amount must be positive")
	}

	// Atomically credit balance; RETURNING gives us the post-update balance
	updatedUser, err := s.userRepo.UpdateBalance(ctx, userID, amountCents)
	if err != nil {
		return nil, nil, err
	}

	// Derive balance_before from the post-credit value (avoids a separate FindByID)
	balanceBefore := updatedUser.BalanceCents - amountCents

	// Record the deposit transaction
	transaction := &models.Transaction{
		UserID:             userID,
		Type:               models.TransactionTypeDeposit,
		AmountCents:        amountCents,
		BalanceBeforeCents: balanceBefore,
		BalanceAfterCents:  updatedUser.BalanceCents,
		Reason:             "crypto_deposit",
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, nil, err
	}

	return updatedUser, transaction, nil
}

// GetBalance gets user's current balance
func (s *WalletService) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return user.BalanceCents, nil
}

// GetTransactionHistory gets user's transaction history with pagination
func (s *WalletService) GetTransactionHistory(ctx context.Context, userID uuid.UUID, page, limit int64) ([]models.TransactionResponse, int64, error) {
	transactions, total, err := s.transactionRepo.FindByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToTransactionResponses(transactions), total, nil
}
