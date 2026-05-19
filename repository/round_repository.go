package repository

import (
	"context"
	"errors"
	"time"

	"aviator-backend/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RoundRepository struct {
	*BaseRepository
}

func NewRoundRepository() *RoundRepository {
	return &RoundRepository{
		BaseRepository: newBase("rounds"),
	}
}

func (r *RoundRepository) Create(ctx context.Context, round *models.Round) error {
	round.StartedAt = time.Now()
	round.TransitionTime = time.Now()

	query := `
		INSERT INTO rounds (round_id, crash_point_x100, server_seed, server_seed_hash, client_seed, nonce, hash, 
		                    status, total_wagered_cents, total_payout_cents, player_count, consecutive_instant_crashes, 
		                    started_at, transition_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`

	err := r.getTx(ctx).QueryRow(ctx, query,
		round.RoundID, round.CrashPointX100, round.ServerSeed, round.ServerSeedHash, round.ClientSeed, round.Nonce, round.Hash,
		round.Status, round.TotalWageredCents, round.TotalPayoutCents, round.PlayerCount, round.ConsecutiveInstantCrashes,
		round.StartedAt, round.TransitionTime,
	).Scan(&round.ID)

	return err
}

func (r *RoundRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Round, error) {
	query := `
		SELECT id, round_id, crash_point_x100, server_seed, server_seed_hash, client_seed, nonce, hash, 
		       status, total_wagered_cents, total_payout_cents, player_count, consecutive_instant_crashes, 
		       started_at, crashed_at, transition_time
		FROM rounds WHERE id = $1`

	var round models.Round
	var crashedAt *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query, id).Scan(
		&round.ID, &round.RoundID, &round.CrashPointX100, &round.ServerSeed, &round.ServerSeedHash, &round.ClientSeed, &round.Nonce, &round.Hash,
		&round.Status, &round.TotalWageredCents, &round.TotalPayoutCents, &round.PlayerCount, &round.ConsecutiveInstantCrashes,
		&round.StartedAt, &crashedAt, &round.TransitionTime,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("round not found")
		}
		return nil, err
	}

	round.CrashedAt = crashedAt
	return &round, nil
}

func (r *RoundRepository) FindByRoundID(ctx context.Context, roundID string) (*models.Round, error) {
	query := `
		SELECT id, round_id, crash_point_x100, server_seed, server_seed_hash, client_seed, nonce, hash, 
		       status, total_wagered_cents, total_payout_cents, player_count, consecutive_instant_crashes, 
		       started_at, crashed_at, transition_time
		FROM rounds WHERE round_id = $1`

	var round models.Round
	var crashedAt *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query, roundID).Scan(
		&round.ID, &round.RoundID, &round.CrashPointX100, &round.ServerSeed, &round.ServerSeedHash, &round.ClientSeed, &round.Nonce, &round.Hash,
		&round.Status, &round.TotalWageredCents, &round.TotalPayoutCents, &round.PlayerCount, &round.ConsecutiveInstantCrashes,
		&round.StartedAt, &crashedAt, &round.TransitionTime,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("round not found")
		}
		return nil, err
	}

	round.CrashedAt = crashedAt
	return &round, nil
}

func (r *RoundRepository) UpdateStatus(ctx context.Context, roundID string, status models.RoundStatus) error {
	query := `UPDATE rounds SET status = $1 WHERE round_id = $2`
	_, err := r.getTx(ctx).Exec(ctx, query, status, roundID)
	return err
}

func (r *RoundRepository) UpdateRunning(ctx context.Context, roundID string, playerCount int, totalWageredCents int64) error {
	query := `
		UPDATE rounds 
		SET status = $1, player_count = $2, total_wagered_cents = $3
		WHERE round_id = $4`

	_, err := r.getTx(ctx).Exec(ctx, query, models.RoundStatusRunning, playerCount, totalWageredCents, roundID)
	return err
}

func (r *RoundRepository) UpdateCrashed(ctx context.Context, roundID string, totalPayoutCents int64) error {
	query := `
		UPDATE rounds 
		SET status = $1, crashed_at = $2, total_payout_cents = $3
		WHERE round_id = $4`

	_, err := r.getTx(ctx).Exec(ctx, query, models.RoundStatusCrashed, time.Now(), totalPayoutCents, roundID)
	return err
}

func (r *RoundRepository) FindHistory(ctx context.Context, page, limit int64) ([]models.Round, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM rounds WHERE status = $1`
	var total int64
	err := r.getTx(ctx).QueryRow(ctx, countQuery, models.RoundStatusCrashed).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, round_id, crash_point_x100, server_seed, server_seed_hash, client_seed, nonce, hash, 
		       status, total_wagered_cents, total_payout_cents, player_count, consecutive_instant_crashes, 
		       started_at, crashed_at, transition_time
		FROM rounds WHERE status = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`

	offset := (page - 1) * limit
	rows, err := r.getTx(ctx).Query(ctx, query, models.RoundStatusCrashed, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rounds []models.Round
	for rows.Next() {
		var round models.Round
		var crashedAt *time.Time

		err := rows.Scan(
			&round.ID, &round.RoundID, &round.CrashPointX100, &round.ServerSeed, &round.ServerSeedHash, &round.ClientSeed, &round.Nonce, &round.Hash,
			&round.Status, &round.TotalWageredCents, &round.TotalPayoutCents, &round.PlayerCount, &round.ConsecutiveInstantCrashes,
			&round.StartedAt, &crashedAt, &round.TransitionTime,
		)
		if err != nil {
			return nil, 0, err
		}

		round.CrashedAt = crashedAt
		rounds = append(rounds, round)
	}

	return rounds, total, nil
}

func (r *RoundRepository) FindAll(ctx context.Context, page, limit int64) ([]models.Round, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM rounds`
	var total int64
	err := r.getTx(ctx).QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, round_id, crash_point_x100, server_seed, server_seed_hash, client_seed, nonce, hash, 
		       status, total_wagered_cents, total_payout_cents, player_count, consecutive_instant_crashes, 
		       started_at, crashed_at, transition_time
		FROM rounds
		ORDER BY started_at DESC
		LIMIT $1 OFFSET $2`

	offset := (page - 1) * limit
	rows, err := r.getTx(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rounds []models.Round
	for rows.Next() {
		var round models.Round
		var crashedAt *time.Time

		err := rows.Scan(
			&round.ID, &round.RoundID, &round.CrashPointX100, &round.ServerSeed, &round.ServerSeedHash, &round.ClientSeed, &round.Nonce, &round.Hash,
			&round.Status, &round.TotalWageredCents, &round.TotalPayoutCents, &round.PlayerCount, &round.ConsecutiveInstantCrashes,
			&round.StartedAt, &crashedAt, &round.TransitionTime,
		)
		if err != nil {
			return nil, 0, err
		}

		round.CrashedAt = crashedAt
		rounds = append(rounds, round)
	}

	return rounds, total, nil
}

func (r *RoundRepository) GetLatestRound(ctx context.Context) (*models.Round, error) {
	query := `
		SELECT id, round_id, crash_point_x100, server_seed, server_seed_hash, client_seed, nonce, hash, 
		       status, total_wagered_cents, total_payout_cents, player_count, consecutive_instant_crashes, 
		       started_at, crashed_at, transition_time
		FROM rounds
		ORDER BY started_at DESC
		LIMIT 1`

	var round models.Round
	var crashedAt *time.Time

	err := r.getTx(ctx).QueryRow(ctx, query).Scan(
		&round.ID, &round.RoundID, &round.CrashPointX100, &round.ServerSeed, &round.ServerSeedHash, &round.ClientSeed, &round.Nonce, &round.Hash,
		&round.Status, &round.TotalWageredCents, &round.TotalPayoutCents, &round.PlayerCount, &round.ConsecutiveInstantCrashes,
		&round.StartedAt, &crashedAt, &round.TransitionTime,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("no rounds found")
		}
		return nil, err
	}

	round.CrashedAt = crashedAt
	return &round, nil
}

func (r *RoundRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_rounds,
			COALESCE(SUM(total_wagered_cents), 0) as total_wagered,
			COALESCE(SUM(total_payout_cents), 0) as total_payout
		FROM rounds
		WHERE status = $1`

	var totalRounds int64
	var totalWagered, totalPayout int64

	err := r.getTx(ctx).QueryRow(ctx, query, models.RoundStatusCrashed).Scan(&totalRounds, &totalWagered, &totalPayout)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_rounds":  totalRounds,
		"total_wagered": totalWagered,
		"total_payout":  totalPayout,
		"house_profit":  totalWagered - totalPayout,
	}, nil
}

func (r *RoundRepository) CountByStatus(ctx context.Context, status models.RoundStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM rounds WHERE status = $1`
	var count int64
	err := r.getTx(ctx).QueryRow(ctx, query, status).Scan(&count)
	return count, err
}

func (r *RoundRepository) DeleteByRoundID(ctx context.Context, roundID string) error {
	query := `DELETE FROM rounds WHERE round_id = $1`
	_, err := r.getTx(ctx).Exec(ctx, query, roundID)
	return err
}
