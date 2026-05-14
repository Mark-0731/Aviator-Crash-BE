package repository

import (
	"context"
	"time"

	"aviator-backend/database"
	"aviator-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	*BaseRepository
	collection *mongo.Collection // Keep for special operations
}

func NewUserRepository() *UserRepository {
	collection := database.DB.Collection("users")
	return &UserRepository{
		BaseRepository: newBase(collection),
		collection:     collection,
	}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	id, err := r.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.BaseRepository.FindByID(ctx, id, &user)
	return &user, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.FindOne(ctx, bson.D{{Key: "email", Value: email}}, &user)
	return &user, err
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.FindOne(ctx, bson.D{{Key: "username", Value: username}}, &user)
	return &user, err
}

// UpdateBalance updates user balance atomically (uses FindOneAndUpdate for atomic operation)
func (r *UserRepository) UpdateBalance(ctx context.Context, userID primitive.ObjectID, amountCents int64) (*models.User, error) {
	var user models.User
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "balance_cents", Value: amountCents}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	return &user, err
}

// DeductBalance deducts balance atomically with balance check
func (r *UserRepository) DeductBalance(ctx context.Context, userID primitive.ObjectID, amountCents int64) (*models.User, error) {
	var user models.User
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: userID},
			{Key: "balance_cents", Value: bson.D{{Key: "$gte", Value: amountCents}}},
			{Key: "is_banned", Value: false},
		},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "balance_cents", Value: -amountCents}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	return &user, err
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID primitive.ObjectID, ip string) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "last_login_ip", Value: ip},
			{Key: "last_login_at", Value: time.Now()},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
}

func (r *UserRepository) BanUser(ctx context.Context, userID primitive.ObjectID, reason string) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "is_banned", Value: true},
			{Key: "ban_reason", Value: reason},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
}

func (r *UserRepository) UnbanUser(ctx context.Context, userID primitive.ObjectID) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "is_banned", Value: false},
			{Key: "ban_reason", Value: ""},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
}

func (r *UserRepository) FindAll(ctx context.Context, page, limit int64, search string) ([]models.User, int64, error) {
	filter := bson.D{}
	if search != "" {
		filter = bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "username", Value: bson.D{{Key: "$regex", Value: search}, {Key: "$options", Value: "i"}}}},
			bson.D{{Key: "email", Value: bson.D{{Key: "$regex", Value: search}, {Key: "$options", Value: "i"}}}},
		}}}
	}

	var users []models.User
	total, err := r.FindWithPagination(ctx, filter, &users, page, limit, "created_at", -1)
	return users, total, err
}

// AdjustBalance adjusts user balance (admin operation)
func (r *UserRepository) AdjustBalance(ctx context.Context, userID primitive.ObjectID, amountCents int64) (*models.User, error) {
	var user models.User
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "balance_cents", Value: amountCents}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	return &user, err
}

func (r *UserRepository) CountByRegistrationIP(ctx context.Context, ip string) (int64, error) {
	return r.CountDocuments(ctx, bson.D{{Key: "registration_ip", Value: ip}})
}

// CountAll returns the total number of users
func (r *UserRepository) CountAll(ctx context.Context) (int64, error) {
	return r.CountDocuments(ctx, bson.D{})
}

// UpdateClientSeed sets a player's custom client seed for provably fair play
func (r *UserRepository) UpdateClientSeed(ctx context.Context, userID primitive.ObjectID, clientSeed string) error {
	return r.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "client_seed", Value: clientSeed},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
}

// GetClientSeedsByIDs returns a map of userID → clientSeed for a slice of IDs
func (r *UserRepository) GetClientSeedsByIDs(ctx context.Context, ids []primitive.ObjectID) ([]string, error) {
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}}
	opts := options.Find().SetProjection(bson.D{{Key: "client_seed", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var seeds []string
	for cursor.Next(ctx) {
		var result struct {
			ClientSeed string `bson:"client_seed"`
		}
		if err := cursor.Decode(&result); err == nil {
			if result.ClientSeed != "" {
				seeds = append(seeds, result.ClientSeed)
			}
		}
	}
	return seeds, nil
}
