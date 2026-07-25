package intelligence

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTunerRepo struct {
	*storage.UnavailableRepository
}

func TestNewAutoTuner(t *testing.T) {
	repo := &mockTunerRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
	}

	tuner := NewAutoTuner(repo)

	require.NotNil(t, tuner)
	assert.Equal(t, repo, tuner.repo)
	assert.Equal(t, 1*time.Minute, tuner.interval)
	assert.NotNil(t, tuner.stop)
}

func TestAutoTuner_StartStop(t *testing.T) {
	repo := &mockTunerRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
	}

	tuner := NewAutoTuner(repo)
	// Override interval for faster test execution
	tuner.interval = 10 * time.Millisecond

	// Start the tuner
	tuner.Start()

	// Let it run for a bit to ensure it hits the tune() function in the loop
	time.Sleep(50 * time.Millisecond)

	// Stop the tuner
	tuner.Stop()

	// Verify that the stop channel is closed
	select {
	case <-tuner.stop:
		// It's closed, all good
	case <-time.After(100 * time.Millisecond):
		t.Fatal("stop channel was not closed")
	}
}
