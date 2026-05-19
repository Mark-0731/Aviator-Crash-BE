package repository

import (
	"context"
	"time"

	"aviator-backend/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// BaseRepository provides common database operations for CockroachDB
type BaseRepository struct {
	tableName string
}

// newBase creates a new base repository
func newBase(tableName string) *BaseRepository {
	return &BaseRepository{tableName: tableName}
}

// Executor interface for both Pool and Tx
type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// getTx returns transaction or pool
func (r *BaseRepository) getTx(ctx context.Context) Executor {
	if tx, ok := ctx.Value("tx").(pgx.Tx); ok {
		return tx
	}
	return database.Pool
}

// DeleteExpired deletes records where expires_at < now
func (r *BaseRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM ` + r.tableName + ` WHERE expires_at < $1`
	result, err := r.getTx(ctx).Exec(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ParseUUID safely parses a UUID string
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// UUIDToString converts UUID to string
func UUIDToString(id uuid.UUID) string {
	return id.String()
}
