package game

import (
	"errors"
	"sync"

	"aviator-backend/models"
)

// GameStateStore defines the interface for game state management
// ARCHITECTURE: Interface-driven design allows future Redis migration without business logic changes
// All methods are pure state operations - NO business logic inside store
type GameStateStore interface {
	SetPhase(phase models.RoundStatus) error
	GetPhase() (models.RoundStatus, error)
	SetCurrentRound(round *models.Round) error
	GetCurrentRound() (*models.Round, error)
	SetActiveBets(bets map[string]*models.Bet) error
	GetActiveBets() (map[string]*models.Bet, error)
	AddActiveBet(userID string, bet *models.Bet) error
	RemoveActiveBet(userID string) error
	GetActiveBet(userID string) (*models.Bet, error)
	IncrementPlayerCount() error
	GetPlayerCount() (int, error)
	SetPlayerCount(count int) error
	ClearActiveBets() error
	SetCurrentMultiplier(multiplier float64) error
	GetCurrentMultiplier() (float64, error)
}

// InMemoryGameState implements GameStateStore using in-memory storage
// MIGRATION PATH: Replace with RedisGameState for distributed deployment
type InMemoryGameState struct {
	phase             models.RoundStatus
	currentRound      *models.Round
	activeBets        map[string]*models.Bet // key: userID
	playerCount       int
	currentMultiplier float64
	mu                sync.RWMutex
}

// NewInMemoryGameState creates a new in-memory game state store
func NewInMemoryGameState() *InMemoryGameState {
	return &InMemoryGameState{
		phase:      models.RoundStatusWaiting,
		activeBets: make(map[string]*models.Bet),
	}
}

func (s *InMemoryGameState) SetPhase(phase models.RoundStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phase = phase
	return nil
}

func (s *InMemoryGameState) GetPhase() (models.RoundStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phase, nil
}

func (s *InMemoryGameState) SetCurrentRound(round *models.Round) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentRound = round
	return nil
}

func (s *InMemoryGameState) GetCurrentRound() (*models.Round, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentRound == nil {
		return nil, errors.New("no current round")
	}
	// Return a copy to prevent external modification of shared state
	roundCopy := *s.currentRound
	return &roundCopy, nil
}

func (s *InMemoryGameState) SetActiveBets(bets map[string]*models.Bet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeBets = bets
	return nil
}

func (s *InMemoryGameState) GetActiveBets() (map[string]*models.Bet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to prevent external modification
	betsCopy := make(map[string]*models.Bet, len(s.activeBets))
	for k, v := range s.activeBets {
		betsCopy[k] = v
	}
	return betsCopy, nil
}

func (s *InMemoryGameState) AddActiveBet(userID string, bet *models.Bet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeBets[userID] = bet
	return nil
}

func (s *InMemoryGameState) RemoveActiveBet(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeBets, userID)
	return nil
}

func (s *InMemoryGameState) GetActiveBet(userID string) (*models.Bet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bet, exists := s.activeBets[userID]
	if !exists {
		return nil, errors.New("no active bet for user")
	}
	return bet, nil
}

func (s *InMemoryGameState) IncrementPlayerCount() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playerCount++
	return nil
}

func (s *InMemoryGameState) GetPlayerCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playerCount, nil
}

func (s *InMemoryGameState) SetPlayerCount(count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playerCount = count
	return nil
}

func (s *InMemoryGameState) ClearActiveBets() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeBets = make(map[string]*models.Bet)
	s.playerCount = 0
	return nil
}

func (s *InMemoryGameState) SetCurrentMultiplier(multiplier float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentMultiplier = multiplier
	return nil
}

func (s *InMemoryGameState) GetCurrentMultiplier() (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentMultiplier, nil
}

// Redis Migration Guide:
//
// type RedisGameState struct {
//     client *redis.Client
// }
//
// func (s *RedisGameState) SetPhase(phase models.RoundStatus) error {
//     return s.client.Set(ctx, "game:phase", string(phase), 0).Err()
// }
//
// func (s *RedisGameState) GetPhase() (models.RoundStatus, error) {
//     val, err := s.client.Get(ctx, "game:phase").Result()
//     return models.RoundStatus(val), err
// }
//
// func (s *RedisGameState) SetCurrentRound(round *models.Round) error {
//     data, _ := json.Marshal(round)
//     return s.client.Set(ctx, "game:current_round", data, 0).Err()
// }
//
// func (s *RedisGameState) AddActiveBet(userID string, bet *models.Bet) error {
//     data, _ := json.Marshal(bet)
//     return s.client.HSet(ctx, "game:active_bets", userID, data).Err()
// }
//
// ... implement remaining methods following same pattern
