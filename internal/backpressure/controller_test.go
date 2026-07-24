package backpressure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestController_Stats(t *testing.T) {
	config := DefaultConfig()
	c := NewController(config)

	t.Run("Initial Stats", func(t *testing.T) {
		stats := c.Stats()
		assert.Equal(t, int64(0), stats.TotalRequests)
		assert.Equal(t, int64(0), stats.AcceptedRequests)
		assert.Equal(t, int64(0), stats.RejectedRequests)
		assert.Equal(t, int64(0), stats.QueuedRequests)
		assert.Equal(t, int64(0), stats.CircuitOpens)
		assert.Equal(t, int64(0), stats.CircuitCloses)
		assert.Equal(t, time.Duration(0), stats.AvgQueueWaitTime)
		assert.Equal(t, 0, stats.CurrentQueueDepth)
		assert.Equal(t, 0, stats.CurrentConcurrency)
		assert.Equal(t, 0.0, stats.ErrorRate)
	})

	t.Run("Stats After Request", func(t *testing.T) {
		ctx := context.Background()
		release, err := c.Acquire(ctx)
		require.NoError(t, err)

		stats := c.Stats()
		assert.Equal(t, int64(1), stats.TotalRequests)
		assert.Equal(t, int64(1), stats.AcceptedRequests)
		assert.Equal(t, int64(0), stats.RejectedRequests)
		assert.Equal(t, 1, stats.CurrentConcurrency)

		release()
		stats = c.Stats()
		assert.Equal(t, 0, stats.CurrentConcurrency)
	})

	t.Run("Stats With Error Rate", func(t *testing.T) {
		c.RecordFailure()
		stats := c.Stats()
		assert.Greater(t, stats.ErrorRate, 0.0)

		c.RecordSuccess(time.Millisecond)
		stats2 := c.Stats()
		assert.Less(t, stats2.ErrorRate, stats.ErrorRate)
	})
}
