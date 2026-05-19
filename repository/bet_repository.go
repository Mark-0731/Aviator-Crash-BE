package repository

import (
	"context"
	"errors"
	"time"

	"aviator-backend/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BetRepository struct {
	*BaseRepository
}

func NewBetRepository() *BetRepository {
	return &BetRepository{
		BaseRepository: newBase("bets"),
	}
}

func (r *BetRepository) Create(ctx context.Context, bet *models.Bet) error {
	bet.PlacedAt = time.Now()

	query := `
		INSERT INTO bets (user_id, round_id, amount_cents, profit_cents, status, placed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err := r.getTx(ctx).QueryRow(ctx, query,
		bet.UserID, bet.RoundID, bet.AmountCents, bet.ProfitCents, bet.Status, bet.PlacedAt,
	).Scan(&bet.ID)

	return err
}

func (r *BetRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Bet, error) {
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE id = $1`

	var bet models.Bet
	var cashedOutAt *time.Time
	var cashoutMultiplier *int64

	err := r.getTx(ctx).QueryRow(ctx, query, id).Scan(
		&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
		&bet.Status, &bet.PlacedAt, &cashedOutAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("bet not found")
		}
		return nil, err
	}

	bet.CashoutMultiplierX100 = cashoutMultiplier
	bet.CashedOutAt = cashedOutAt

	return &bet, nil
}

func (r *BetRepository) FindByUserAndRound(ctx context.Context, userID uuid.UUID, roundID string) (*models.Bet, error) {
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE user_id = $1 AND round_id = $2`

	var bet models.Bet
	var cashedOutAt *time.Time
	var cashoutMultiplier *int64

	err := r.getTx(ctx).QueryRow(ctx, query, userID, roundID).Scan(
		&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
		&bet.Status, &bet.PlacedAt, &cashedOutAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("bet not found")
		}
		return nil, err
	}

	bet.CashoutMultiplierX100 = cashoutMultiplier
	bet.CashedOutAt = cashedOutAt

	return &bet, nil
}

func (r *BetRepository) FindByUserAndRoundAll(ctx context.Context, userID uuid.UUID, roundID string) ([]models.Bet, error) {
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE user_id = $1 AND round_id = $2`

	rows, err := r.getTx(ctx).Query(ctx, query, userID, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []models.Bet
	for rows.Next() {
		var bet models.Bet
		var cashedOutAt *time.Time
		var cashoutMultiplier *int64

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
			&bet.Status, &bet.PlacedAt, &cashedOutAt,
		)
		if err != nil {
			return nil, err
		}

		bet.CashoutMultiplierX100 = cashoutMultiplier
		bet.CashedOutAt = cashedOutAt
		bets = append(bets, bet)
	}

	return bets, nil
}

func (r *BetRepository) FindPendingByRound(ctx context.Context, roundID string) ([]models.Bet, error) {
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE round_id = $1 AND status = $2`

	rows, err := r.getTx(ctx).Query(ctx, query, roundID, models.BetStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []models.Bet
	for rows.Next() {
		var bet models.Bet
		var cashedOutAt *time.Time
		var cashoutMultiplier *int64

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
			&bet.Status, &bet.PlacedAt, &cashedOutAt,
		)
		if err != nil {
			return nil, err
		}

		bet.CashoutMultiplierX100 = cashoutMultiplier
		bet.CashedOutAt = cashedOutAt
		bets = append(bets, bet)
	}

	return bets, nil
}

func (r *BetRepository) FindAllPending(ctx context.Context) ([]models.Bet, error) {
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE status = $1`

	rows, err := r.getTx(ctx).Query(ctx, query, models.BetStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []models.Bet
	for rows.Next() {
		var bet models.Bet
		var cashedOutAt *time.Time
		var cashoutMultiplier *int64

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
			&bet.Status, &bet.PlacedAt, &cashedOutAt,
		)
		if err != nil {
			return nil, err
		}

		bet.CashoutMultiplierX100 = cashoutMultiplier
		bet.CashedOutAt = cashedOutAt
		bets = append(bets, bet)
	}

	return bets, nil
}

func (r *BetRepository) UpdateCashout(ctx context.Context, betID uuid.UUID, multiplierX100 int64, profitCents int64) error {
	query := `
		UPDATE bets 
		SET cashout_multiplier_x100 = $1, profit_cents = $2, status = $3, cashed_out_at = $4
		WHERE id = $5 AND status = $6`

	result, err := r.getTx(ctx).Exec(ctx, query,
		multiplierX100, profitCents, models.BetStatusWon, time.Now(), betID, models.BetStatusPending,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		// Return pgx.ErrNoRows so the service layer can detect already-settled bets
		// via `err == pgx.ErrNoRows` (consistent with other repo methods)
		return pgx.ErrNoRows
	}

	return nil
}

func (r *BetRepository) UpdateStatus(ctx context.Context, betID uuid.UUID, status models.BetStatus, profitCents int64) error {
	query := `UPDATE bets SET status = $1, profit_cents = $2 WHERE id = $3`
	_, err := r.getTx(ctx).Exec(ctx, query, status, profitCents, betID)
	return err
}

func (r *BetRepository) BulkUpdateLost(ctx context.Context, bets []models.Bet) error {
	if len(bets) == 0 {
		return nil
	}

	// Execute all updates within the same transaction obtained from context
	tx := r.getTx(ctx)
	for _, bet := range bets {
		if _, err := tx.Exec(ctx,
			`UPDATE bets SET status = $1, profit_cents = $2 WHERE id = $3`,
			models.BetStatusLost, -bet.AmountCents, bet.ID,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *BetRepository) FindWonByRound(ctx context.Context, roundID string) ([]models.Bet, error) {
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE round_id = $1 AND status = $2`

	rows, err := r.getTx(ctx).Query(ctx, query, roundID, models.BetStatusWon)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []models.Bet
	for rows.Next() {
		var bet models.Bet
		var cashedOutAt *time.Time
		var cashoutMultiplier *int64

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
			&bet.Status, &bet.PlacedAt, &cashedOutAt,
		)
		if err != nil {
			return nil, err
		}

		bet.CashoutMultiplierX100 = cashoutMultiplier
		bet.CashedOutAt = cashedOutAt
		bets = append(bets, bet)
	}

	return bets, nil
}

func (r *BetRepository) FindByUser(ctx context.Context, userID uuid.UUID, page, limit int64) ([]models.Bet, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM bets WHERE user_id = $1`
	var total int64
	err := r.getTx(ctx).QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE user_id = $1
		ORDER BY placed_at DESC
		LIMIT $2 OFFSET $3`

	offset := (page - 1) * limit
	rows, err := r.getTx(ctx).Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var bets []models.Bet
	for rows.Next() {
		var bet models.Bet
		var cashedOutAt *time.Time
		var cashoutMultiplier *int64

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
			&bet.Status, &bet.PlacedAt, &cashedOutAt,
		)
		if err != nil {
			return nil, 0, err
		}

		bet.CashoutMultiplierX100 = cashoutMultiplier
		bet.CashedOutAt = cashedOutAt
		bets = append(bets, bet)
	}

	return bets, total, nil
}

func (r *BetRepository) FindByRound(ctx context.Context, roundID string) ([]models.Bet, error) {
	query := `
		SELECT id, user_id, round_id, amount_cents, cashout_multiplier_x100, profit_cents, 
		       status, placed_at, cashed_out_at
		FROM bets WHERE round_id = $1`

	rows, err := r.getTx(ctx).Query(ctx, query, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bets []models.Bet
	for rows.Next() {
		var bet models.Bet
		var cashedOutAt *time.Time
		var cashoutMultiplier *int64

		err := rows.Scan(
			&bet.ID, &bet.UserID, &bet.RoundID, &bet.AmountCents, &cashoutMultiplier, &bet.ProfitCents,
			&bet.Status, &bet.PlacedAt, &cashedOutAt,
		)
		if err != nil {
			return nil, err
		}

		bet.CashoutMultiplierX100 = cashoutMultiplier
		bet.CashedOutAt = cashedOutAt
		bets = append(bets, bet)
	}

	return bets, nil
}

func (r *BetRepository) CountByRound(ctx context.Context, roundID string) (int64, error) {
	query := `SELECT COUNT(*) FROM bets WHERE round_id = $1`
	var count int64
	err := r.getTx(ctx).QueryRow(ctx, query, roundID).Scan(&count)
	return count, err
}

func (r *BetRepository) GetTotalWageredByRound(ctx context.Context, roundID string) (int64, error) {
	query := `SELECT COALESCE(SUM(amount_cents), 0) FROM bets WHERE round_id = $1`
	var total int64
	err := r.getTx(ctx).QueryRow(ctx, query, roundID).Scan(&total)
	return total, err
}

func (r *BetRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM bets WHERE id = $1`
	_, err := r.getTx(ctx).Exec(ctx, query, id)
	return err
}
