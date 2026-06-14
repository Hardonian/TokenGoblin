package backpressure

import (
	"context"
	"sync"
	"time"
)

// Controller manages backpressure across the system
type Controller struct {
	mu          sync.RWMutex
	limiter     *RateLimiter
	circuitOpen bool
	lastReject  time.Time
	config      Config
	stats       Stats
}

// Config holds backpressure configuration
type Config struct {
	MaxConcurrentRequests int
	MaxQueueSize          int
	RateLimitRPS          int
	EnableCircuitBreaker  bool
	CircuitOpenThreshold  float64 // Error rate to open circuit (0.0-1.0)
	CircuitCloseAfter     time.Duration
	RejectTimeout         time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxConcurrentRequests: 1000,
		MaxQueueSize:          10000,
		RateLimitRPS:          10000,
		EnableCircuitBreaker:  true,
		CircuitOpenThreshold:  0.5, // 50% error rate
		CircuitCloseAfter:     30 * time.Second,
		RejectTimeout:         5 * time.Second,
	}
}

// Stats tracks backpressure metrics
type Stats struct {
	TotalRequests       int64
	AcceptedRequests    int64
	RejectedRequests    int64
	QueuedRequests      int64
	CircuitOpens        int64
	CircuitCloses       int64
	AvgQueueWaitTime    time.Duration
	CurrentQueueDepth   int
	CurrentConcurrency  int
	ErrorRate           float64
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	maxTokens float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps int) *RateLimiter {
	return &RateLimiter{
		tokens:     float64(rps),
		maxTokens:  float64(rps),
		refillRate: float64(rps),
		lastRefill: time.Now(),
	}
}

// Allow checks if a request can proceed
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens = min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now

	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available or context is cancelled
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// NewController creates a new backpressure controller
func NewController(config Config) *Controller {
	return &Controller{
		config:  config,
		limiter: NewRateLimiter(config.RateLimitRPS),
	}
}

// Acquire attempts to acquire permission to process a request
func (c *Controller) Acquire(ctx context.Context) (ReleaseFunc, error) {
	c.mu.RLock()
	circuitOpen := c.circuitOpen
	c.mu.RUnlock()

	if circuitOpen {
		c.stats.RejectedRequests++
		return nil, ErrCircuitOpen
	}

	// Check rate limit
	if !c.limiter.Allow() {
		// Try waiting with timeout
		waitCtx, cancel := context.WithTimeout(ctx, c.config.RejectTimeout)
		defer cancel()
		if err := c.limiter.Wait(waitCtx); err != nil {
			c.stats.RejectedRequests++
			return nil, ErrRateLimited
		}
	}

	c.stats.TotalRequests++
	c.stats.AcceptedRequests++
	c.mu.Lock()
	c.stats.CurrentConcurrency++
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		c.stats.CurrentConcurrency--
		c.mu.Unlock()
	}, nil
}

// RecordSuccess records a successful request
func (c *Controller) RecordSuccess(latency time.Duration) {
	c.updateErrorRate(false)
}

// RecordFailure records a failed request
func (c *Controller) RecordFailure() {
	c.updateErrorRate(true)
}

// updateErrorRate updates the circuit breaker state based on error rate
func (c *Controller) updateErrorRate(isError bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple exponential moving average for error rate
	alpha := 0.1
	if isError {
		c.stats.ErrorRate = c.stats.ErrorRate*(1-alpha) + alpha
	} else {
		c.stats.ErrorRate = c.stats.ErrorRate * (1 - alpha)
	}

	if c.config.EnableCircuitBreaker && !c.circuitOpen {
		if c.stats.ErrorRate >= c.config.CircuitOpenThreshold {
			c.circuitOpen = true
			c.lastReject = time.Now()
			c.stats.CircuitOpens++
			// Schedule circuit close
			go func() {
				time.Sleep(c.config.CircuitCloseAfter)
				c.mu.Lock()
				if c.circuitOpen && time.Since(c.lastReject) > c.config.CircuitCloseAfter {
					c.circuitOpen = false
					c.stats.CircuitCloses++
				}
				c.mu.Unlock()
			}()
		}
	}
}

// Stats returns current statistics
func (c *Controller) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// ReleaseFunc is called when a request completes
type ReleaseFunc func()

// Errors
var (
	ErrCircuitOpen   = &BackpressureError{Code: "CIRCUIT_OPEN", Message: "Circuit breaker is open"}
	ErrRateLimited   = &BackpressureError{Code: "RATE_LIMITED", Message: "Request rate limit exceeded"}
	ErrQueueFull     = &BackpressureError{Code: "QUEUE_FULL", Message: "Request queue is full"}
	ErrTimeout       = &BackpressureError{Code: "TIMEOUT", Message: "Request timed out waiting for capacity"}
)

type BackpressureError struct {
	Code    string
	Message string
}

func (e *BackpressureError) Error() string {
	return e.Message
}

// Health checks the backpressure controller state
func (c *Controller) Health(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.circuitOpen {
		return ErrCircuitOpen
	}
	if c.stats.CurrentConcurrency > c.config.MaxConcurrentRequests {
		return ErrQueueFull
	}
	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}