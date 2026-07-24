package backpressure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 1000, config.MaxConcurrentRequests, "Expected MaxConcurrentRequests to be 1000")
	assert.Equal(t, 10000, config.MaxQueueSize, "Expected MaxQueueSize to be 10000")
	assert.Equal(t, 10000, config.RateLimitRPS, "Expected RateLimitRPS to be 10000")
	assert.Equal(t, true, config.EnableCircuitBreaker, "Expected EnableCircuitBreaker to be true")
	assert.Equal(t, 0.5, config.CircuitOpenThreshold, "Expected CircuitOpenThreshold to be 0.5")
	assert.Equal(t, 30*time.Second, config.CircuitCloseAfter, "Expected CircuitCloseAfter to be 30 seconds")
	assert.Equal(t, 5*time.Second, config.RejectTimeout, "Expected RejectTimeout to be 5 seconds")
}
