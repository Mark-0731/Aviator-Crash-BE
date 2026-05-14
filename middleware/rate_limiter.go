package middleware

import (
	"sync"
	"time"

	"aviator-backend/config"
	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS
// Token bucket rate limiter implementation

// RateLimiter stores rate limiters per IP
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

var (
	restLimiter     *RateLimiter
	authLimiter     *RateLimiter
	once            sync.Once
	cleanupRestOnce sync.Once
	cleanupAuthOnce sync.Once
)

func initLimiters() {
	once.Do(func() {
		// Initialize rate limiters
		restLimiter = NewRateLimiter(rate.Limit(config.AppConfig.RateLimitREST)/60.0, config.AppConfig.RateLimitREST)
		authLimiter = NewRateLimiter(rate.Limit(config.AppConfig.RateLimitAuth)/60.0, config.AppConfig.RateLimitAuth)
	})
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// getLimiter gets or creates a limiter for an IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// cleanup removes old limiters (run periodically)
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Remove limiters that haven't been used recently
	for ip, limiter := range rl.limiters {
		if limiter.Tokens() == float64(rl.burst) {
			delete(rl.limiters, ip)
		}
	}
}

// startCleanupRoutine starts a periodic cleanup goroutine for a rate limiter
func startCleanupRoutine(limiter *RateLimiter) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()
}

// RateLimitREST applies rate limiting to REST endpoints
func RateLimitREST() gin.HandlerFunc {
	initLimiters()
	cleanupRestOnce.Do(func() { startCleanupRoutine(restLimiter) })

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := restLimiter.getLimiter(ip)

		if !limiter.Allow() {
			utils.RespondWithError(c, utils.ErrRateLimited, "Too many requests, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitAuth applies stricter rate limiting to auth endpoints
func RateLimitAuth() gin.HandlerFunc {
	initLimiters()
	cleanupAuthOnce.Do(func() { startCleanupRoutine(authLimiter) })

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := authLimiter.getLimiter(ip)

		if !limiter.Allow() {
			utils.RespondWithError(c, utils.ErrRateLimited, "Too many authentication attempts, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}

// WSRateLimiter for WebSocket connections (per-connection)
type WSRateLimiter struct {
	limiter *rate.Limiter
}

// NewWSRateLimiter creates a WebSocket rate limiter
func NewWSRateLimiter() *WSRateLimiter {
	return &WSRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(config.AppConfig.RateLimitWS), config.AppConfig.RateLimitWS),
	}
}

// Allow checks if a message is allowed
func (wrl *WSRateLimiter) Allow() bool {
	return wrl.limiter.Allow()
}
