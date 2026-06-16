package client

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter provides thread-safe rate limiting for HTTP requests.
type RateLimiter struct {
	limiter *rate.Limiter
	mu      sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the specified requests per second.
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond),
	}
}

// Wait blocks until the rate limiter allows another request.
// It respects context cancellation.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.limiter.Wait(ctx)
}

// SetRate updates the rate limit to a new requests per second value.
func (rl *RateLimiter) SetRate(requestsPerSecond int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limiter.SetLimit(rate.Limit(requestsPerSecond))
	rl.limiter.SetBurst(requestsPerSecond)
}

// Made with Bob
