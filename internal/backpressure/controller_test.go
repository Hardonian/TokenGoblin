package backpressure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestController_Acquire(t *testing.T) {
	t.Run("circuit_open", func(t *testing.T) {
		cfg := DefaultConfig()
		c := NewController(cfg)

		// Manually set circuit to open
		c.mu.Lock()
		c.circuitOpen = true
		c.mu.Unlock()

		release, err := c.Acquire(context.Background())

		assert.ErrorIs(t, err, ErrCircuitOpen)
		assert.Nil(t, release)

		stats := c.Stats()
		assert.Equal(t, int64(1), stats.RejectedRequests)
	})

	t.Run("rate_limited", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.RateLimitRPS = 1
		cfg.RejectTimeout = 5 * time.Millisecond
		c := NewController(cfg)

		// Exhaust the 1 token immediately
		release, err := c.Acquire(context.Background())
		require.NoError(t, err)
		defer release()

		// Attempt to acquire next token, which should fail due to rate limit and RejectTimeout
		start := time.Now()
		release2, err := c.Acquire(context.Background())

		assert.ErrorIs(t, err, ErrRateLimited)
		assert.Nil(t, release2)
		assert.GreaterOrEqual(t, time.Since(start), 5*time.Millisecond)

		stats := c.Stats()
		assert.Equal(t, int64(1), stats.RejectedRequests)
	})

	t.Run("happy_path", func(t *testing.T) {
		cfg := DefaultConfig()
		c := NewController(cfg)

		release, err := c.Acquire(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, release)

		stats := c.Stats()
		assert.Equal(t, int64(1), stats.TotalRequests)
		assert.Equal(t, int64(1), stats.AcceptedRequests)
		assert.Equal(t, 1, stats.CurrentConcurrency)

		// Execute release function and verify concurrency stat decreases
		release()

		stats = c.Stats()
		assert.Equal(t, 0, stats.CurrentConcurrency)
	})
}

func TestController_RecordSuccess(t *testing.T) {
	cfg := DefaultConfig()
	c := NewController(cfg)

	c.stats.ErrorRate = 0.5

	c.RecordSuccess(100 * time.Millisecond)

	stats := c.Stats()
	assert.InDelta(t, 0.45, stats.ErrorRate, 0.0001)
}

func TestController_RecordFailure(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CircuitOpenThreshold = 0.5
	cfg.EnableCircuitBreaker = true
	c := NewController(cfg)

	c.stats.ErrorRate = 0.45

	c.RecordFailure()

	stats := c.Stats()
	assert.InDelta(t, 0.505, stats.ErrorRate, 0.0001)

	c.mu.RLock()
	assert.True(t, c.circuitOpen)
	c.mu.RUnlock()
}

func TestController_Health(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		cfg := DefaultConfig()
		c := NewController(cfg)

		err := c.Health(context.Background())
		assert.NoError(t, err)
	})

	t.Run("circuit_open", func(t *testing.T) {
		cfg := DefaultConfig()
		c := NewController(cfg)

		c.mu.Lock()
		c.circuitOpen = true
		c.mu.Unlock()

		err := c.Health(context.Background())
		assert.ErrorIs(t, err, ErrCircuitOpen)
	})

	t.Run("queue_full", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxConcurrentRequests = 5
		c := NewController(cfg)

		c.mu.Lock()
		c.stats.CurrentConcurrency = 10
		c.mu.Unlock()

		err := c.Health(context.Background())
		assert.ErrorIs(t, err, ErrQueueFull)
	})
}

func TestBackpressureError_Error(t *testing.T) {
	err := &BackpressureError{Code: "TEST", Message: "Test message"}
	assert.Equal(t, "Test message", err.Error())
}
