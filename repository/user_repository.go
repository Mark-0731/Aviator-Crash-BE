package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"aviator-backend/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	*BaseRepository
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		BaseRepository: newBase("users"),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (username, email, password, balance_cents, client_seed, is_admin, is_banned, registration_ip, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	err := r.getTx(ctx).QueryRow(ctx, query,
		user.Username, user.Email, user.Password, user.BalanceCents, user.ClientSeed,
		user.IsAdmin, user.IsBanned, user.RegistrationIP, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)

	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, email, password, balance_cents, client_seed, is_admin, is_banned, 
		       ban_reason, registration_ip, last_login_ip, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`

	var user models.User
	var banReason, lastLoginIP *string
	var lastLoginTime *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.BalanceCents, &user.ClientSeed,
		&user.IsAdmin, &user.IsBanned, &banReason, &user.RegistrationIP, &lastLoginIP, &lastLoginTime,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if banReason != nil {
		user.BanReason = *banReason
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginTime != nil {
		user.LastLoginAt = *lastLoginTime
	}

	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password, balance_cents, client_seed, is_admin, is_banned, 
		       ban_reason, registration_ip, last_login_ip, last_login_at, created_at, updated_at
		FROM users WHERE email = $1`

	var user models.User
	var banReason, lastLoginIP *string
	var lastLoginTime *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.BalanceCents, &user.ClientSeed,
		&user.IsAdmin, &user.IsBanned, &banReason, &user.RegistrationIP, &lastLoginIP, &lastLoginTime,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if banReason != nil {
		user.BanReason = *banReason
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginTime != nil {
		user.LastLoginAt = *lastLoginTime
	}

	return &user, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password, balance_cents, client_seed, is_admin, is_banned, 
		       ban_reason, registration_ip, last_login_ip, last_login_at, created_at, updated_at
		FROM users WHERE username = $1`

	var user models.User
	var banReason, lastLoginIP *string
	var lastLoginTime *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.BalanceCents, &user.ClientSeed,
		&user.IsAdmin, &user.IsBanned, &banReason, &user.RegistrationIP, &lastLoginIP, &lastLoginTime,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if banReason != nil {
		user.BanReason = *banReason
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginTime != nil {
		user.LastLoginAt = *lastLoginTime
	}

	return &user, nil
}

func (r *UserRepository) UpdateBalance(ctx context.Context, userID uuid.UUID, amountCents int64) (*models.User, error) {
	query := `
		UPDATE users 
		SET balance_cents = balance_cents + $1, updated_at = $2
		WHERE id = $3
		RETURNING id, username, email, password, balance_cents, client_seed, is_admin, is_banned, 
		          ban_reason, registration_ip, last_login_ip, last_login_at, created_at, updated_at`

	var user models.User
	var banReason, lastLoginIP *string
	var lastLoginTime *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query, amountCents, time.Now(), userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.BalanceCents, &user.ClientSeed,
		&user.IsAdmin, &user.IsBanned, &banReason, &user.RegistrationIP, &lastLoginIP, &lastLoginTime,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if banReason != nil {
		user.BanReason = *banReason
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginTime != nil {
		user.LastLoginAt = *lastLoginTime
	}

	return &user, nil
}

func (r *UserRepository) DeductBalance(ctx context.Context, userID uuid.UUID, amountCents int64) (*models.User, error) {
	query := `
		UPDATE users 
		SET balance_cents = balance_cents - $1, updated_at = $2
		WHERE id = $3 AND balance_cents >= $1 AND is_banned = false
		RETURNING id, username, email, password, balance_cents, client_seed, is_admin, is_banned, 
		          ban_reason, registration_ip, last_login_ip, last_login_at, created_at, updated_at`

	var user models.User
	var banReason, lastLoginIP *string
	var lastLoginTime *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query, amountCents, time.Now(), userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.BalanceCents, &user.ClientSeed,
		&user.IsAdmin, &user.IsBanned, &banReason, &user.RegistrationIP, &lastLoginIP, &lastLoginTime,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("insufficient balance or user banned")
		}
		return nil, err
	}

	if banReason != nil {
		user.BanReason = *banReason
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginTime != nil {
		user.LastLoginAt = *lastLoginTime
	}

	return &user, nil
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID, ip string) error {
	query := `UPDATE users SET last_login_ip = $1, last_login_at = $2, updated_at = $3 WHERE id = $4`
	_, err := r.getTx(ctx).Exec(ctx, query, ip, time.Now(), time.Now(), userID)
	return err
}

func (r *UserRepository) BanUser(ctx context.Context, userID uuid.UUID, reason string) error {
	query := `UPDATE users SET is_banned = true, ban_reason = $1, updated_at = $2 WHERE id = $3`
	_, err := r.getTx(ctx).Exec(ctx, query, reason, time.Now(), userID)
	return err
}

func (r *UserRepository) UnbanUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET is_banned = false, ban_reason = '', updated_at = $1 WHERE id = $2`
	_, err := r.getTx(ctx).Exec(ctx, query, time.Now(), userID)
	return err
}

func (r *UserRepository) FindAll(ctx context.Context, page, limit int64, search string) ([]models.User, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM users`
	args := []interface{}{}
	argPos := 1

	if search != "" {
		// fmt.Sprintf is required — string(rune(N)) yields control chars, not "1", "2", …
		countQuery += fmt.Sprintf(` WHERE username ILIKE $%d OR email ILIKE $%d`, argPos, argPos)
		args = append(args, "%"+search+"%")
		argPos++
	}

	var total int64
	err := r.getTx(ctx).QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, username, email, password, balance_cents, client_seed, is_admin, is_banned,
		       ban_reason, registration_ip, last_login_ip, last_login_at, created_at, updated_at
		FROM users`

	if search != "" {
		query += fmt.Sprintf(` WHERE username ILIKE $%d OR email ILIKE $%d`, 1, 1)
	}

	offset := (page - 1) * limit
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.getTx(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var banReason, lastLoginIP *string
		var lastLoginTime *time.Time

		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.Password, &user.BalanceCents, &user.ClientSeed,
			&user.IsAdmin, &user.IsBanned, &banReason, &user.RegistrationIP, &lastLoginIP, &lastLoginTime,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if banReason != nil {
			user.BanReason = *banReason
		}
		if lastLoginIP != nil {
			user.LastLoginIP = *lastLoginIP
		}
		if lastLoginTime != nil {
			user.LastLoginAt = *lastLoginTime
		}

		users = append(users, user)
	}

	return users, total, nil
}

func (r *UserRepository) AdjustBalance(ctx context.Context, userID uuid.UUID, amountCents int64) (*models.User, error) {
	return r.UpdateBalance(ctx, userID, amountCents)
}

func (r *UserRepository) CountByRegistrationIP(ctx context.Context, ip string) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE registration_ip = $1`
	var count int64
	err := r.getTx(ctx).QueryRow(ctx, query, ip).Scan(&count)
	return count, err
}

func (r *UserRepository) CountAll(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int64
	err := r.getTx(ctx).QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *UserRepository) UpdateClientSeed(ctx context.Context, userID uuid.UUID, clientSeed string) error {
	query := `UPDATE users SET client_seed = $1, updated_at = $2 WHERE id = $3`
	_, err := r.getTx(ctx).Exec(ctx, query, clientSeed, time.Now(), userID)
	return err
}

func (r *UserRepository) GetClientSeedsByIDs(ctx context.Context, ids []uuid.UUID) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}

	query := `SELECT client_seed FROM users WHERE id = ANY($1) AND client_seed != ''`
	rows, err := r.getTx(ctx).Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seeds []string
	for rows.Next() {
		var seed string
		if err := rows.Scan(&seed); err == nil && seed != "" {
			seeds = append(seeds, seed)
		}
	}

	return seeds, nil
}

func (r *UserRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.getTx(ctx).Exec(ctx, query, id)
	return err
}
