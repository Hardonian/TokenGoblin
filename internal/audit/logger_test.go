package audit

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	*storage.UnavailableRepository
	savedEvent *domain.AuditEvent
	saveErr    error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
	}
}

func (m *mockRepo) SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	m.savedEvent = &event
	return m.saveErr
}

func TestNewLogger(t *testing.T) {
	repo := newMockRepo()
	logger := NewLogger(repo, nil)

	assert.NotNil(t, logger)
	assert.Equal(t, repo, logger.repo)
	assert.Equal(t, slog.Default(), logger.logger)
}

func TestLogger_LogEvent_Success(t *testing.T) {
	repo := newMockRepo()
	logger := NewLogger(repo, nil)

	event := Event{
		ID:       "test-id",
		TenantID: "tenant-1",
		Type:     EventAPIKeyCreated,
		Actor:    "user-1",
		Metadata: map[string]interface{}{"key": "value"},
		Timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	err := logger.LogEvent(context.Background(), event)
	assert.NoError(t, err)
	assert.NotNil(t, repo.savedEvent)
	assert.Equal(t, "test-id", repo.savedEvent.EventID)
	assert.Equal(t, "tenant-1", repo.savedEvent.TenantID)
	assert.Equal(t, string(EventAPIKeyCreated), repo.savedEvent.Type)
	assert.Equal(t, "user-1", repo.savedEvent.Actor)
	assert.Equal(t, event.Metadata, repo.savedEvent.Metadata)
	assert.Equal(t, event.Timestamp, repo.savedEvent.Timestamp)
}

func TestLogger_LogEvent_GeneratedFields(t *testing.T) {
	repo := newMockRepo()
	logger := NewLogger(repo, nil)

	event := Event{
		TenantID: "tenant-1",
		Type:     EventAPIKeyCreated,
		Actor:    "user-1",
	}

	err := logger.LogEvent(context.Background(), event)
	assert.NoError(t, err)
	assert.NotNil(t, repo.savedEvent)
	assert.NotEmpty(t, repo.savedEvent.EventID)
	assert.Contains(t, repo.savedEvent.EventID, "aud_")
	assert.False(t, repo.savedEvent.Timestamp.IsZero())
}

func TestLogger_LogEvent_Error(t *testing.T) {
	repo := newMockRepo()
	expectedErr := errors.New("database error")
	repo.saveErr = expectedErr
	logger := NewLogger(repo, nil)

	event := Event{
		TenantID: "tenant-1",
		Type:     EventAPIKeyCreated,
		Actor:    "user-1",
	}

	err := logger.LogEvent(context.Background(), event)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
