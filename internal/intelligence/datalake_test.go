package intelligence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDatalakeRepo struct {
	*storage.UnavailableRepository
	getUnexportedEvents func(ctx context.Context, limit int) ([]domain.TokenEvent, error)
	markEventsExported  func(ctx context.Context, eventIDs []string) error
}

func (m *mockDatalakeRepo) GetUnexportedEvents(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
	if m.getUnexportedEvents != nil {
		return m.getUnexportedEvents(ctx, limit)
	}
	return nil, nil
}

func (m *mockDatalakeRepo) MarkEventsExported(ctx context.Context, eventIDs []string) error {
	if m.markEventsExported != nil {
		return m.markEventsExported(ctx, eventIDs)
	}
	return nil
}

func TestDataLakeExporter_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()

	events := []domain.TokenEvent{
		{EventID: "evt-1", TenantID: "t-1", ModelID: "gpt-4"},
		{EventID: "evt-2", TenantID: "t-1", ModelID: "gpt-4"},
	}

	marked := false
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEvents: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return events, nil
		},
		markEventsExported: func(ctx context.Context, eventIDs []string) error {
			assert.ElementsMatch(t, []string{"evt-1", "evt-2"}, eventIDs)
			marked = true
			return nil
		},
	}

	exporter := NewDataLakeExporter(repo, tmpDir)

	// Test the exportBatch directly
	exporter.exportBatch()

	assert.True(t, marked, "Expected events to be marked as exported")

	// Check file creation
	dateStr := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(tmpDir, "token_events_"+dateStr+".jsonl")

	info, err := os.Stat(expectedFile)
	require.NoError(t, err)
	assert.True(t, info.Size() > 0)

	// Read back lines
	data, err := os.ReadFile(expectedFile)
	require.NoError(t, err)

	lines := 0
	for _, char := range string(data) {
		if char == '\n' {
			lines++
		}
	}
	assert.Equal(t, 2, lines, "Expected 2 lines in JSONL file")
}

func TestDataLakeExporter_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()

	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEvents: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return nil, nil
		},
		markEventsExported: func(ctx context.Context, eventIDs []string) error {
			t.Fatal("markEventsExported should not be called")
			return nil
		},
	}

	exporter := NewDataLakeExporter(repo, tmpDir)
	exporter.exportBatch()

	// Ensure no files were created
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDataLakeExporter_ReadError(t *testing.T) {
	tmpDir := t.TempDir()

	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEvents: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return nil, errors.New("db error")
		},
		markEventsExported: func(ctx context.Context, eventIDs []string) error {
			t.Fatal("markEventsExported should not be called")
			return nil
		},
	}

	exporter := NewDataLakeExporter(repo, tmpDir)
	// Shouldn't panic
	exporter.exportBatch()
}

func TestDataLakeExporter_WriteError(t *testing.T) {
	tmpDir := t.TempDir()

	events := []domain.TokenEvent{
		{EventID: "evt-1"},
	}

	markedCalled := false
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEvents: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return events, nil
		},
		markEventsExported: func(ctx context.Context, eventIDs []string) error {
			markedCalled = true
			return errors.New("write error")
		},
	}

	exporter := NewDataLakeExporter(repo, tmpDir)
	exporter.exportBatch()

	assert.True(t, markedCalled)
}

func TestDataLakeExporter_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEvents: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return nil, nil
		},
	}

	exporter := NewDataLakeExporter(repo, tmpDir)
	// interval is normally 10s. For tests we can't easily change it without modifying the struct (which has unexported fields)
	// but we can just Start and Stop it to ensure no panics and that Stop closes cleanly.

	exporter.Start()
	// Let the goroutine spin up
	time.Sleep(10 * time.Millisecond)
	exporter.Stop()
	// Let the goroutine shut down
	time.Sleep(10 * time.Millisecond)

	// If it hasn't panicked or deadlocked, test passes.
}
