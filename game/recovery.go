package game

import (
	"context"
	"sync"
	"time"

	"aviator-backend/database"
	"aviator-backend/models"

	"github.com/rs/zerolog/log"
)

// runRecovery handles server restart recovery with retry logic
func (e *Engine) runRecovery() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pendingBets, err := e.findPendingBetsWithRetry(ctx)
	if err != nil {
		log.Error().Err(err).Msg("recovery_failed_to_find_pending_bets")
		return err
	}

	if len(pendingBets) == 0 {
		log.Info().Msg("recovery_complete_no_pending_bets")
		return nil
	}

	log.Info().Int("count", len(pendingBets)).Msg("recovery_started")

	stats := e.refundPendingBets(ctx, pendingBets)

	log.Info().
		Int("total_bets", len(pendingBets)).
		Int("refunds_issued", stats.successCount).
		Int("refunds_failed", stats.failCount).
		Int64("total_refunded_cents", stats.totalRefunded).
		Msg("recovery_complete")

	return nil
}

// recoveryStats holds recovery statistics
type recoveryStats struct {
	totalRefunded int64
	successCount  int
	failCount     int
}

// findPendingBetsWithRetry finds pending bets with retry logic
func (e *Engine) findPendingBetsWithRetry(ctx context.Context) ([]models.Bet, error) {
	var pendingBets []models.Bet
	var err error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		pendingBets, err = e.betRepo.FindAllPending(ctx)
		if err == nil {
			return pendingBets, nil
		}

		if attempt < MaxRetries-1 {
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Msg("recovery_find_pending_retry")
			time.Sleep(RetryDelay * time.Duration(attempt+1))
		}
	}

	return nil, err
}

// refundPendingBets refunds all pending bets in parallel
func (e *Engine) refundPendingBets(ctx context.Context, bets []models.Bet) recoveryStats {
	stats := recoveryStats{}
	semaphore := make(chan struct{}, 10) // Max 10 concurrent refunds
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range bets {
		wg.Add(1)
		go func(bet *models.Bet) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := e.refundBetWithRetry(ctx, bet); err != nil {
				log.Error().
					Err(err).
					Str("bet_id", bet.ID.String()).
					Str("user_id", bet.UserID.String()).
					Msg("refund_failed")
				mu.Lock()
				stats.failCount++
				mu.Unlock()
			} else {
				mu.Lock()
				stats.totalRefunded += bet.AmountCents
				stats.successCount++
				mu.Unlock()
			}
		}(&bets[i])
	}

	wg.Wait()
	return stats
}

// refundBetWithRetry refunds a bet with retry logic
func (e *Engine) refundBetWithRetry(ctx context.Context, bet *models.Bet) error {
	for attempt := 0; attempt < MaxRetries; attempt++ {
		if err := e.refundBet(ctx, bet); err == nil {
			return nil
		} else if attempt < MaxRetries-1 {
			time.Sleep(RetryDelay * time.Duration(attempt+1))
		} else {
			return err
		}
	}
	return nil
}

// refundBet refunds a single bet using a transaction to prevent double-refund
func (e *Engine) refundBet(ctx context.Context, bet *models.Bet) error {
	// Use a CockroachDB transaction to ensure all 3 operations succeed or all fail
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Create context with transaction
	txCtx := context.WithValue(ctx, "tx", tx)

	// Refund balance atomically — RETURNING gives us post-credit balance
	updatedUser, err := e.userRepo.UpdateBalance(txCtx, bet.UserID, bet.AmountCents)
	if err != nil {
		return err
	}

	// Derive balance_before from post-credit value — no separate FindByID needed
	balanceBefore := updatedUser.BalanceCents - bet.AmountCents

	// Update bet status to refunded
	if err := e.betRepo.UpdateStatus(txCtx, bet.ID, models.BetStatusRefunded, 0); err != nil {
		return err
	}

	// Create transaction record
	transaction := &models.Transaction{
		UserID:             bet.UserID,
		Type:               models.TransactionTypeRefund,
		AmountCents:        bet.AmountCents,
		RoundID:            &bet.RoundID,
		BalanceBeforeCents: balanceBefore,
		BalanceAfterCents:  updatedUser.BalanceCents,
		Reason:             "server_restart_recovery",
	}

	if err := e.transactionRepo.Create(txCtx, transaction); err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit(ctx)
}
