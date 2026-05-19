package database

import (
	"context"

	"github.com/rs/zerolog/log"
)

// createTables creates all necessary tables
func createTables(ctx context.Context) error {
	queries := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			balance_cents BIGINT NOT NULL DEFAULT 0,
			client_seed VARCHAR(255) NOT NULL DEFAULT 'default',
			is_admin BOOLEAN NOT NULL DEFAULT false,
			is_banned BOOLEAN NOT NULL DEFAULT false,
			ban_reason TEXT,
			registration_ip VARCHAR(45),
			last_login_ip VARCHAR(45),
			last_login_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Rounds table
		`CREATE TABLE IF NOT EXISTS rounds (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			round_id VARCHAR(255) UNIQUE NOT NULL,
			crash_point_x100 BIGINT NOT NULL,
			server_seed VARCHAR(255) NOT NULL,
			server_seed_hash VARCHAR(255) NOT NULL,
			client_seed VARCHAR(255) NOT NULL,
			nonce BIGINT NOT NULL,
			hash VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			total_wagered_cents BIGINT NOT NULL DEFAULT 0,
			total_payout_cents BIGINT NOT NULL DEFAULT 0,
			player_count INT NOT NULL DEFAULT 0,
			consecutive_instant_crashes INT NOT NULL DEFAULT 0,
			started_at TIMESTAMP NOT NULL DEFAULT NOW(),
			crashed_at TIMESTAMP,
			transition_time TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Bets table
		`CREATE TABLE IF NOT EXISTS bets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			round_id VARCHAR(255) NOT NULL,
			amount_cents BIGINT NOT NULL,
			cashout_multiplier_x100 BIGINT,
			profit_cents BIGINT NOT NULL DEFAULT 0,
			status VARCHAR(50) NOT NULL,
			placed_at TIMESTAMP NOT NULL DEFAULT NOW(),
			cashed_out_at TIMESTAMP
		)`,

		// Transactions table
		`CREATE TABLE IF NOT EXISTS transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			amount_cents BIGINT NOT NULL,
			round_id VARCHAR(255),
			balance_before_cents BIGINT NOT NULL,
			balance_after_cents BIGINT NOT NULL,
			reason TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Refresh tokens table
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			token VARCHAR(500) UNIQUE NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// WebSocket tickets table
		`CREATE TABLE IF NOT EXISTS ws_tickets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			ticket VARCHAR(500) UNIQUE NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			used BOOLEAN NOT NULL DEFAULT false,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Payments table
		`CREATE TABLE IF NOT EXISTS payments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			payment_id VARCHAR(255) UNIQUE NOT NULL,
			order_id VARCHAR(255) UNIQUE NOT NULL,
			payment_status VARCHAR(50) NOT NULL,
			pay_address VARCHAR(255),
			price_amount DECIMAL(20, 8),
			price_currency VARCHAR(10),
			pay_amount DECIMAL(20, 8),
			pay_currency VARCHAR(10),
			amount_received DECIMAL(20, 8),
			purchase_amount_cents BIGINT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		if _, err := Pool.Exec(ctx, query); err != nil {
			log.Error().Err(err).Str("query", query[:50]+"...").Msg("failed_to_create_table")
			return err
		}
	}

	log.Info().Msg("tables_created_successfully")
	return nil
}
