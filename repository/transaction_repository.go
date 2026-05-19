package repository

import (
	"context"
	"errors"
	"time"

	"aviator-backend/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TransactionRepository struct {
	*BaseRepository
}

func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{
		BaseRepository: newBase("transactions"),
	}
}

func (r *TransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	transaction.CreatedAt = time.Now()

	query := `
		INSERT INTO transactions (user_id, type, amount_cents, round_id, balance_before_cents, balance_after_cents, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := r.getTx(ctx).QueryRow(ctx, query,
		transaction.UserID, transaction.Type, transaction.AmountCents, transaction.RoundID,
		transaction.BalanceBeforeCents, transaction.BalanceAfterCents, transaction.Reason, transaction.CreatedAt,
	).Scan(&transaction.ID)

	return err
}

func (r *TransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, type, amount_cents, round_id, balance_before_cents, balance_after_cents, reason, created_at
		FROM transactions WHERE id = $1`

	var transaction models.Transaction
	var roundID *string
	var reason *string

	err := r.getTx(ctx).QueryRow(ctx, query, id).Scan(
		&transaction.ID, &transaction.UserID, &transaction.Type, &transaction.AmountCents, &roundID,
		&transaction.BalanceBeforeCents, &transaction.BalanceAfterCents, &reason, &transaction.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, err
	}

	transaction.RoundID = roundID
	if reason != nil {
		transaction.Reason = *reason
	}

	return &transaction, nil
}

func (r *TransactionRepository) FindByUser(ctx context.Context, userID uuid.UUID, page, limit int64) ([]models.Transaction, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM transactions WHERE user_id = $1`
	var total int64
	err := r.getTx(ctx).QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, user_id, type, amount_cents, round_id, balance_before_cents, balance_after_cents, reason, created_at
		FROM transactions WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	offset := (page - 1) * limit
	rows, err := r.getTx(ctx).Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var roundID *string
		var reason *string

		err := rows.Scan(
			&transaction.ID, &transaction.UserID, &transaction.Type, &transaction.AmountCents, &roundID,
			&transaction.BalanceBeforeCents, &transaction.BalanceAfterCents, &reason, &transaction.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		transaction.RoundID = roundID
		if reason != nil {
			transaction.Reason = *reason
		}

		transactions = append(transactions, transaction)
	}

	return transactions, total, nil
}

func (r *TransactionRepository) FindByRound(ctx context.Context, roundID string) ([]models.Transaction, error) {
	query := `
		SELECT id, user_id, type, amount_cents, round_id, balance_before_cents, balance_after_cents, reason, created_at
		FROM transactions WHERE round_id = $1`

	rows, err := r.getTx(ctx).Query(ctx, query, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var roundIDPtr *string
		var reason *string

		err := rows.Scan(
			&transaction.ID, &transaction.UserID, &transaction.Type, &transaction.AmountCents, &roundIDPtr,
			&transaction.BalanceBeforeCents, &transaction.BalanceAfterCents, &reason, &transaction.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		transaction.RoundID = roundIDPtr
		if reason != nil {
			transaction.Reason = *reason
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *TransactionRepository) FindByType(ctx context.Context, transactionType models.TransactionType, page, limit int64) ([]models.Transaction, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM transactions WHERE type = $1`
	var total int64
	err := r.getTx(ctx).QueryRow(ctx, countQuery, transactionType).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, user_id, type, amount_cents, round_id, balance_before_cents, balance_after_cents, reason, created_at
		FROM transactions WHERE type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	offset := (page - 1) * limit
	rows, err := r.getTx(ctx).Query(ctx, query, transactionType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var roundID *string
		var reason *string

		err := rows.Scan(
			&transaction.ID, &transaction.UserID, &transaction.Type, &transaction.AmountCents, &roundID,
			&transaction.BalanceBeforeCents, &transaction.BalanceAfterCents, &reason, &transaction.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		transaction.RoundID = roundID
		if reason != nil {
			transaction.Reason = *reason
		}

		transactions = append(transactions, transaction)
	}

	return transactions, total, nil
}

func (r *TransactionRepository) GetTotalByType(ctx context.Context, transactionType models.TransactionType) (int64, error) {
	query := `SELECT COALESCE(SUM(amount_cents), 0) FROM transactions WHERE type = $1`
	var total int64
	err := r.getTx(ctx).QueryRow(ctx, query, transactionType).Scan(&total)
	return total, err
}

func (r *TransactionRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT type, COUNT(*) as count, COALESCE(SUM(amount_cents), 0) as total
		FROM transactions
		WHERE user_id = $1
		GROUP BY type`

	rows, err := r.getTx(ctx).Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	for rows.Next() {
		var transactionType string
		var count int64
		var total int64

		err := rows.Scan(&transactionType, &count, &total)
		if err != nil {
			return nil, err
		}

		stats[transactionType] = map[string]interface{}{
			"count": count,
			"total": total,
		}
	}

	return stats, nil
}

func (r *TransactionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM transactions WHERE user_id = $1`
	_, err := r.getTx(ctx).Exec(ctx, query, userID)
	return err
}
