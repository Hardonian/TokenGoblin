package billing

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/stretchr/testify/assert"
)

// mockRepository is a dummy repository to satisfy storage.Repository for tests.
type mockRepository struct {
	*storage.UnavailableRepository
}

func TestStripeSyncer_ContextCancellation(t *testing.T) {
	repo := &mockRepository{UnavailableRepository: storage.NewUnavailableRepository(nil)}
	syncer := NewStripeSyncer(repo, nil)

	// Context already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		syncer.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// success, it returned immediately
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestStripeSyncer_TickerMechanics(t *testing.T) {
	repo := &mockRepository{UnavailableRepository: storage.NewUnavailableRepository(nil)}
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	syncer := NewStripeSyncer(repo, logger).WithInterval(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		syncer.Start(ctx)
		close(done)
	}()

	<-done

	// Verify that the ticker triggered at least once by checking the logs
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Starting Stripe usage sync")
	assert.Contains(t, logOutput, "Stripe usage sync completed")
}
