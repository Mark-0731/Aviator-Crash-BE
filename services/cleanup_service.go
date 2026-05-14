package services

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CleanupService handles periodic cleanup tasks
type CleanupService struct {
	authService *AuthService
	ticker      *time.Ticker
	stopChan    chan struct{}
	stopOnce    sync.Once
}

// NewCleanupService creates a new cleanup service
func NewCleanupService() *CleanupService {
	return &CleanupService{
		authService: NewAuthService(),
		ticker:      time.NewTicker(1 * time.Hour),
		stopChan:    make(chan struct{}),
	}
}

// Start begins the cleanup job
func (s *CleanupService) Start() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.runCleanup()
			case <-s.stopChan:
				s.ticker.Stop()
				return
			}
		}
	}()
	log.Info().Msg("cleanup_service_started")
}

// Stop stops the cleanup job
func (s *CleanupService) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
		log.Info().Msg("cleanup_service_stopped")
	})
}

// runCleanup performs the cleanup operation
func (s *CleanupService) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.authService.CleanupExpiredTokens(ctx); err != nil {
		log.Error().Err(err).Msg("cleanup_job_failed")
	} else {
		log.Debug().Msg("cleanup_job_completed")
	}
}
