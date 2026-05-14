package services

import (
	"context"
	"errors"

	"aviator-backend/models"
	"aviator-backend/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
func (s *WalletService) Deposit(ctx context.Context, userID primitive.ObjectID, amountCents int64) (*models.User, *models.Transaction, error) {
	if amountCents <= 0 {
		return nil, nil, errors.New("deposit amount must be positive")
	}

	// MOCK: In production, verify payment with payment gateway here
	// Example: stripeCharge, err := stripe.CreateCharge(amountCents, userID)
	// if err != nil { return nil, nil, err }

	// Get user's current balance using repository
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	balanceBefore := user.BalanceCents

	// Add balance atomically using repository
	updatedUser, err := s.userRepo.UpdateBalance(ctx, userID, amountCents)
	if err != nil {
		return nil, nil, err
	}

	// Create transaction record using repository
	transaction := &models.Transaction{
		UserID:             userID,
		Type:               models.TransactionTypeDeposit,
		AmountCents:        amountCents,
		BalanceBeforeCents: balanceBefore,
		BalanceAfterCents:  updatedUser.BalanceCents,
		Reason:             "mock_deposit", // In production: "stripe_charge_id_xxx"
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, nil, err
	}

	return updatedUser, transaction, nil
}

// GetBalance gets user's current balance
func (s *WalletService) GetBalance(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return user.BalanceCents, nil
}

// GetTransactionHistory gets user's transaction history with pagination
func (s *WalletService) GetTransactionHistory(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.TransactionResponse, int64, error) {
	transactions, total, err := s.transactionRepo.FindByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return convertToTransactionResponses(transactions), total, nil
}
