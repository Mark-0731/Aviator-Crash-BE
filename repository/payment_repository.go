package repository

import (
	"context"
	"errors"
	"time"

	"aviator-backend/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PaymentRepository struct {
	*BaseRepository
}

func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{
		BaseRepository: newBase("payments"),
	}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *models.CryptoPayment) error {
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()

	query := `
		INSERT INTO payments (user_id, payment_id, order_id, payment_status, pay_address, price_amount, price_currency, 
		                      pay_amount, pay_currency, amount_received, purchase_amount_cents, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	err := r.getTx(ctx).QueryRow(ctx, query,
		payment.UserID, payment.PaymentID, payment.OrderID, payment.Status, payment.PayAddress,
		payment.PriceAmountUSD, "USD", payment.PayAmount, payment.PayCurrency, payment.ActuallyPaid,
		payment.CreditedCents, payment.CreatedAt, payment.UpdatedAt,
	).Scan(&payment.ID)

	return err
}

func (r *PaymentRepository) FindByPaymentID(ctx context.Context, paymentID string) (*models.CryptoPayment, error) {
	query := `
		SELECT id, user_id, payment_id, order_id, payment_status, pay_address, price_amount, price_currency,
		       pay_amount, pay_currency, amount_received, purchase_amount_cents, created_at, updated_at
		FROM payments WHERE payment_id = $1`

	var payment models.CryptoPayment
	var priceCurrency string // DB column exists but not on model; scanned and discarded

	err := r.getTx(ctx).QueryRow(ctx, query, paymentID).Scan(
		&payment.ID, &payment.UserID, &payment.PaymentID, &payment.OrderID, &payment.Status, &payment.PayAddress,
		&payment.PriceAmountUSD, &priceCurrency, &payment.PayAmount, &payment.PayCurrency, &payment.ActuallyPaid,
		&payment.CreditedCents, &payment.CreatedAt, &payment.UpdatedAt,
	)
	_ = priceCurrency // intentionally discarded — not on CryptoPayment model

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("payment not found")
		}
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) FindByOrderID(ctx context.Context, orderID string) (*models.CryptoPayment, error) {
	query := `
		SELECT id, user_id, payment_id, order_id, payment_status, pay_address, price_amount, price_currency,
		       pay_amount, pay_currency, amount_received, purchase_amount_cents, created_at, updated_at
		FROM payments WHERE order_id = $1`

	var payment models.CryptoPayment
	var priceCurrency string // DB column exists but not on model; scanned and discarded

	err := r.getTx(ctx).QueryRow(ctx, query, orderID).Scan(
		&payment.ID, &payment.UserID, &payment.PaymentID, &payment.OrderID, &payment.Status, &payment.PayAddress,
		&payment.PriceAmountUSD, &priceCurrency, &payment.PayAmount, &payment.PayCurrency, &payment.ActuallyPaid,
		&payment.CreditedCents, &payment.CreatedAt, &payment.UpdatedAt,
	)
	_ = priceCurrency // intentionally discarded — not on CryptoPayment model

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("payment not found")
		}
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) FindByPaymentIDAndUser(ctx context.Context, paymentID string, userID uuid.UUID) (*models.CryptoPayment, error) {
	query := `
		SELECT id, user_id, payment_id, order_id, payment_status, pay_address, price_amount, price_currency,
		       pay_amount, pay_currency, amount_received, purchase_amount_cents, created_at, updated_at
		FROM payments WHERE payment_id = $1 AND user_id = $2`

	var payment models.CryptoPayment
	var priceCurrency string // DB column exists but not on model; scanned and discarded

	err := r.getTx(ctx).QueryRow(ctx, query, paymentID, userID).Scan(
		&payment.ID, &payment.UserID, &payment.PaymentID, &payment.OrderID, &payment.Status, &payment.PayAddress,
		&payment.PriceAmountUSD, &priceCurrency, &payment.PayAmount, &payment.PayCurrency, &payment.ActuallyPaid,
		&payment.CreditedCents, &payment.CreatedAt, &payment.UpdatedAt,
	)
	_ = priceCurrency // intentionally discarded — not on CryptoPayment model

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("payment not found")
		}
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, paymentID string, status models.PaymentStatus, actuallyPaid float64, creditedCents int64) error {
	query := `
		UPDATE payments 
		SET payment_status = $1, amount_received = $2, purchase_amount_cents = $3, updated_at = $4
		WHERE payment_id = $5`

	_, err := r.getTx(ctx).Exec(ctx, query, status, actuallyPaid, creditedCents, time.Now(), paymentID)
	return err
}

func (r *PaymentRepository) UpdateStatusByOrderID(ctx context.Context, orderID string, status models.PaymentStatus, actuallyPaid float64, creditedCents int64) error {
	query := `
		UPDATE payments 
		SET payment_status = $1, amount_received = $2, purchase_amount_cents = $3, updated_at = $4
		WHERE order_id = $5`

	_, err := r.getTx(ctx).Exec(ctx, query, status, actuallyPaid, creditedCents, time.Now(), orderID)
	return err
}

// IsAlreadyFinished checks if a payment is in a terminal finished state.
// Uses a direct COUNT query to avoid fragile error-string matching.
func (r *PaymentRepository) IsAlreadyFinished(ctx context.Context, paymentID string) (bool, error) {
	var count int
	err := r.getTx(ctx).QueryRow(ctx,
		`SELECT COUNT(*) FROM payments WHERE payment_id = $1 AND payment_status = $2`,
		paymentID, models.PaymentStatusFinished,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsAlreadyFinishedByOrderID checks if a payment identified by order_id is already finished.
// Uses a direct COUNT query to avoid fragile error-string matching.
func (r *PaymentRepository) IsAlreadyFinishedByOrderID(ctx context.Context, orderID string) (bool, error) {
	var count int
	err := r.getTx(ctx).QueryRow(ctx,
		`SELECT COUNT(*) FROM payments WHERE order_id = $1 AND payment_status = $2`,
		orderID, models.PaymentStatusFinished,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
