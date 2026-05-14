package game

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
	"sync"
	"time"

	"aviator-backend/models"
	"aviator-backend/repository"

	"github.com/rs/zerolog/log"
)

const (
	BetGraceWindow = 200 * time.Millisecond
	MaxRetries     = 3
	RetryDelay     = 100 * time.Millisecond
)

// Engine manages the game loop and state transitions
type Engine struct {
	state                     GameStateStore
	hub                       BroadcastHub
	userRepo                  *repository.UserRepository
	betRepo                   *repository.BetRepository
	roundRepo                 *repository.RoundRepository
	transactionRepo           *repository.TransactionRepository
	consecutiveInstantCrashes int
	roundStartTime            time.Time
	currentRound              *models.Round
	nextServerSeed            string // Pre-committed seed for the upcoming round
	stopChan                  chan struct{}
	stoppedChan               chan struct{}
	stopOnce                  sync.Once
	mu                        sync.RWMutex
	rng                       *mathrand.Rand
}

// BroadcastHub interface for WebSocket broadcasting
type BroadcastHub interface {
	Broadcast(event string, data interface{})
	GetConnectedUserIDs() []string
}

// NewEngine creates a new game engine with cryptographically secure RNG
func NewEngine(state GameStateStore, hub BroadcastHub) *Engine {
	var seed int64
	binary.Read(rand.Reader, binary.BigEndian, &seed)

	return &Engine{
		state:           state,
		hub:             hub,
		userRepo:        repository.NewUserRepository(),
		betRepo:         repository.NewBetRepository(),
		roundRepo:       repository.NewRoundRepository(),
		transactionRepo: repository.NewTransactionRepository(),
		stopChan:        make(chan struct{}),
		stoppedChan:     make(chan struct{}),
		rng:             mathrand.New(mathrand.NewSource(seed)),
	}
}

// Start begins the game loop with panic recovery
func (e *Engine) Start() {
	log.Info().Msg("game_engine_starting")

	if err := e.runRecovery(); err != nil {
		log.Error().Err(err).Msg("recovery_failed")
	}

	go func() {
		defer close(e.stoppedChan)
		for {
			select {
			case <-e.stopChan:
				log.Info().Msg("game_engine_stopping")
				return
			default:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Error().Interface("panic", r).Msg("game_engine_panic")
							time.Sleep(1 * time.Second)
						}
					}()
					e.runGameLoop()
				}()
			}
		}
	}()
}

// Stop gracefully stops the game engine (safe to call multiple times)
func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopChan)
	})
	<-e.stoppedChan
}

// runGameLoop executes one complete game cycle
func (e *Engine) runGameLoop() {
	e.waitingPhase()
	e.runningPhase()
	e.crashedPhase()
}

// settleBets settles all bets for the round
func (e *Engine) settleBets(ctx context.Context, round *models.Round) int64 {
	// Get all pending bets
	pendingBets, err := e.betRepo.FindPendingByRound(ctx, round.RoundID)
	if err != nil {
		log.Error().Err(err).Msg("failed_to_find_pending_bets")
		return 0
	}

	// Bulk update all losing bets
	if len(pendingBets) > 0 {
		if err := e.betRepo.BulkUpdateLost(ctx, pendingBets); err != nil {
			log.Error().Err(err).Msg("bulk_settle_failed")
		}
	}

	// Calculate total payout (only cashed out bets)
	wonBets, err := e.betRepo.FindWonByRound(ctx, round.RoundID)
	if err != nil {
		log.Error().Err(err).Msg("failed_to_find_won_bets")
		return 0
	}

	totalPayout := int64(0)
	for _, bet := range wonBets {
		totalPayout += bet.AmountCents + bet.ProfitCents
	}

	return totalPayout
}

// IsWithinGraceWindow checks if a bet is within the grace window
func (e *Engine) IsWithinGraceWindow(betPlacedAt time.Time, round *models.Round) bool {
	graceEnd := round.TransitionTime.Add(BetGraceWindow)
	return betPlacedAt.Before(graceEnd) || betPlacedAt.Equal(graceEnd)
}

// GetRoundStartTime returns the current round start time
func (e *Engine) GetRoundStartTime() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.roundStartTime
}
