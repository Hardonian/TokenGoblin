package backpressure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestController_Stats(t *testing.T) {
	config := DefaultConfig()
	c := NewController(config)

	// Initially stats should be zero
	stats := c.Stats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.AcceptedRequests)
	assert.Equal(t, int64(0), stats.RejectedRequests)

	// Simulate some activity to change stats
	ctx := context.Background()
	release, err := c.Acquire(ctx)
	assert.NoError(t, err)

	// Stats should reflect the accepted request
	stats = c.Stats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.AcceptedRequests)
	assert.Equal(t, int(1), stats.CurrentConcurrency)
	assert.Equal(t, int64(0), stats.RejectedRequests)

	// Simulate an error to change error rate
	c.RecordFailure()
	stats = c.Stats()
	assert.Greater(t, stats.ErrorRate, float64(0))

	// Release the request
	release()

	// Stats should reflect the released request
	stats = c.Stats()
	assert.Equal(t, int(0), stats.CurrentConcurrency)
}
