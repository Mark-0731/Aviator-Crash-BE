package database

import (
	"context"

	"github.com/rs/zerolog/log"
)

// createIndexes creates all required indexes for optimal performance
func createIndexes(ctx context.Context) error {
	indexes := []string{
		// Users indexes
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_registration_ip ON users(registration_ip)`,

		// Rounds indexes
		`CREATE INDEX IF NOT EXISTS idx_rounds_round_id ON rounds(round_id)`,
		`CREATE INDEX IF NOT EXISTS idx_rounds_status ON rounds(status)`,
		`CREATE INDEX IF NOT EXISTS idx_rounds_started_at ON rounds(started_at DESC)`,

		// Bets indexes
		`CREATE INDEX IF NOT EXISTS idx_bets_user_id ON bets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bets_round_id ON bets(round_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bets_status ON bets(status)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_bets_user_round ON bets(user_id, round_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bets_placed_at ON bets(placed_at DESC)`,

		// Transactions indexes
		`CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC)`,

		// Refresh tokens indexes
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at)`,

		// WebSocket tickets indexes
		`CREATE INDEX IF NOT EXISTS idx_ws_tickets_ticket ON ws_tickets(ticket)`,
		`CREATE INDEX IF NOT EXISTS idx_ws_tickets_user_id ON ws_tickets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ws_tickets_expires_at ON ws_tickets(expires_at)`,

		// Payments indexes
		`CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_payment_id ON payments(payment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(payment_status)`,
	}

	for _, query := range indexes {
		if _, err := Pool.Exec(ctx, query); err != nil {
			log.Error().Err(err).Str("query", query[:50]+"...").Msg("failed_to_create_index")
			return err
		}
	}

	log.Info().Msg("database_indexes_created")
	return nil
}
