package repository

import (
	"context"
	"time"

	"aviator-backend/database"
	"aviator-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// PaymentRepository handles all DB operations for CryptoPayment
type PaymentRepository struct {
	*BaseRepository
}

func NewPaymentRepository() *PaymentRepository {
	collection := database.DB.Collection("crypto_payments")
	return &PaymentRepository{
		BaseRepository: newBase(collection),
	}
}

// Create inserts a new CryptoPayment document
func (r *PaymentRepository) Create(ctx context.Context, payment *models.CryptoPayment) error {
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()
	id, err := r.InsertOne(ctx, payment)
	if err != nil {
		return err
	}
	payment.ID = id
	return nil
}

// FindByPaymentID finds a payment by its NOWPayments payment_id
func (r *PaymentRepository) FindByPaymentID(ctx context.Context, paymentID string) (*models.CryptoPayment, error) {
	var payment models.CryptoPayment
	err := r.FindOne(ctx, bson.D{{Key: "payment_id", Value: paymentID}}, &payment)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// FindByOrderID finds a payment by its unique order_id (used in webhooks)
func (r *PaymentRepository) FindByOrderID(ctx context.Context, orderID string) (*models.CryptoPayment, error) {
	var payment models.CryptoPayment
	err := r.FindOne(ctx, bson.D{{Key: "order_id", Value: orderID}}, &payment)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// FindByPaymentIDAndUser finds a payment by payment_id scoped to a specific user (prevents leaking other users' payments)
func (r *PaymentRepository) FindByPaymentIDAndUser(ctx context.Context, paymentID string, userID primitive.ObjectID) (*models.CryptoPayment, error) {
	var payment models.CryptoPayment
	err := r.FindOne(ctx, bson.D{
		{Key: "payment_id", Value: paymentID},
		{Key: "user_id", Value: userID},
	}, &payment)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdateStatus updates a payment's status, actually_paid, and credited_cents atomically
func (r *PaymentRepository) UpdateStatus(ctx context.Context, paymentID string, status models.PaymentStatus, actuallyPaid float64, creditedCents int64) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "payment_id", Value: paymentID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "actually_paid", Value: actuallyPaid},
			{Key: "credited_cents", Value: creditedCents},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
}

// UpdateStatusByOrderID updates a payment's status by order_id (used in webhooks)
func (r *PaymentRepository) UpdateStatusByOrderID(ctx context.Context, orderID string, status models.PaymentStatus, actuallyPaid float64, creditedCents int64) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "order_id", Value: orderID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "actually_paid", Value: actuallyPaid},
			{Key: "credited_cents", Value: creditedCents},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
}

// IsAlreadyFinished returns true if the payment has already been credited (idempotency guard)
func (r *PaymentRepository) IsAlreadyFinished(ctx context.Context, paymentID string) (bool, error) {
	payment, err := r.FindByPaymentID(ctx, paymentID)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return payment.Status == models.PaymentStatusFinished, nil
}

// IsAlreadyFinishedByOrderID returns true if the payment has already been credited by order_id
func (r *PaymentRepository) IsAlreadyFinishedByOrderID(ctx context.Context, orderID string) (bool, error) {
	payment, err := r.FindByOrderID(ctx, orderID)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return payment.Status == models.PaymentStatusFinished, nil
}
