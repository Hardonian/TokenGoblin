package backpressure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestController_Stats(t *testing.T) {
	config := DefaultConfig()
	c := NewController(config)

	// Set some stats directly under the lock
	expectedStats := Stats{
		TotalRequests:      100,
		AcceptedRequests:   90,
		RejectedRequests:   10,
		QueuedRequests:     5,
		CircuitOpens:       2,
		CircuitCloses:      1,
		AvgQueueWaitTime:   10 * time.Millisecond,
		CurrentQueueDepth:  3,
		CurrentConcurrency: 50,
		ErrorRate:          0.15,
	}

	c.mu.Lock()
	c.stats = expectedStats
	c.mu.Unlock()

	// Call the method under test
	actualStats := c.Stats()

	// Assert equality
	assert.Equal(t, expectedStats, actualStats, "Stats() should return a copy of the current statistics")
}
