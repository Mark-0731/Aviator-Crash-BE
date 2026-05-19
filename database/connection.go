package database

import (
	"context"
	"time"

	"aviator-backend/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var (
	Pool *pgxpool.Pool
)

// ConnectCockroachDB establishes CockroachDB connection with connection pooling
func ConnectCockroachDB(ctx context.Context) error {
	poolConfig, err := pgxpool.ParseConfig(config.AppConfig.DatabaseURL)
	if err != nil {
		return err
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(config.AppConfig.DBMaxPool)
	poolConfig.MinConns = int32(config.AppConfig.DBMinPool)
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}

	// Ping to verify connection
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	Pool = pool

	log.Info().
		Str("database", "cockroachdb").
		Int32("max_pool", poolConfig.MaxConns).
		Int32("min_pool", poolConfig.MinConns).
		Msg("db_connected")

	// Create tables
	if err := createTables(ctx); err != nil {
		return err
	}

	// Create indexes
	if err := createIndexes(ctx); err != nil {
		return err
	}

	return nil
}

// DisconnectCockroachDB closes the CockroachDB connection
func DisconnectCockroachDB() {
	if Pool != nil {
		Pool.Close()
		log.Info().Msg("db_disconnected")
	}
}
