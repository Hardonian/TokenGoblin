package backpressure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestController_Stats(t *testing.T) {
	config := DefaultConfig()
	c := NewController(config)

	// Verify initial stats are all zeros
	initialStats := c.Stats()
	assert.Equal(t, int64(0), initialStats.TotalRequests)
	assert.Equal(t, int64(0), initialStats.AcceptedRequests)
	assert.Equal(t, int64(0), initialStats.RejectedRequests)
	assert.Equal(t, int64(0), initialStats.CircuitOpens)
	assert.Equal(t, float64(0), initialStats.ErrorRate)
	assert.Equal(t, 0, initialStats.CurrentConcurrency)

	// Make a request and verify stats update
	release, err := c.Acquire(context.Background())
	assert.NoError(t, err)

	activeStats := c.Stats()
	assert.Equal(t, int64(1), activeStats.TotalRequests)
	assert.Equal(t, int64(1), activeStats.AcceptedRequests)
	assert.Equal(t, 1, activeStats.CurrentConcurrency)

	// Release and verify concurrency drops
	release()

	finalStats := c.Stats()
	assert.Equal(t, 0, finalStats.CurrentConcurrency)
	assert.Equal(t, int64(1), finalStats.TotalRequests)

	// Verify RecordFailure updates error rate
	c.RecordFailure()
	failStats := c.Stats()
	assert.Greater(t, failStats.ErrorRate, float64(0))

	// Verify Circuit opens after error threshold
	// Mock a high error rate by calling RecordFailure multiple times
	for i := 0; i < 10; i++ {
		c.RecordFailure()
	}

	openCircuitStats := c.Stats()
	assert.Equal(t, int64(1), openCircuitStats.CircuitOpens)

	// Try acquiring when circuit is open
	_, err = c.Acquire(context.Background())
	assert.ErrorIs(t, err, ErrCircuitOpen)

	rejectStats := c.Stats()
	assert.Equal(t, int64(1), rejectStats.RejectedRequests)

}
