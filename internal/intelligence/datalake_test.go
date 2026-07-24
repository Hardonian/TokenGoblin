package intelligence

import (
	"bufio"
	"context"
	"encoding/json"
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
	getUnexportedEventsFn func(ctx context.Context, limit int) ([]domain.TokenEvent, error)
	markEventsExportedFn  func(ctx context.Context, eventIDs []string) error
}

func (m *mockDatalakeRepo) GetUnexportedEvents(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
	if m.getUnexportedEventsFn != nil {
		return m.getUnexportedEventsFn(ctx, limit)
	}
	return nil, m.UnavailableRepository.Cause
}

func (m *mockDatalakeRepo) MarkEventsExported(ctx context.Context, eventIDs []string) error {
	if m.markEventsExportedFn != nil {
		return m.markEventsExportedFn(ctx, eventIDs)
	}
	return m.UnavailableRepository.Cause
}

func TestNewDataLakeExporter(t *testing.T) {
	repo := &mockDatalakeRepo{UnavailableRepository: storage.NewUnavailableRepository(nil)}

	// Test with explicit sinkDir
	sinkDir := t.TempDir()
	exporter := NewDataLakeExporter(repo, sinkDir)
	assert.NotNil(t, exporter)
	assert.Equal(t, sinkDir, exporter.sinkDir)
	assert.Equal(t, 100, exporter.batchLimit)
	assert.Equal(t, 10*time.Second, exporter.interval)

	// Test with default sinkDir (needs to clean up created dir afterwards)
	defer os.RemoveAll("./data/datalake")
	exporterDefault := NewDataLakeExporter(repo, "")
	assert.NotNil(t, exporterDefault)
	assert.Equal(t, "./data/datalake", exporterDefault.sinkDir)

	// Ensure default directory was created
	info, err := os.Stat("./data/datalake")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDataLakeExporter_exportBatch_Success(t *testing.T) {
	sinkDir := t.TempDir()
	events := []domain.TokenEvent{
		{EventID: "evt-1", TenantID: "tenant-1"},
		{EventID: "evt-2", TenantID: "tenant-2"},
	}

	markedIDs := []string{}
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEventsFn: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return events, nil
		},
		markEventsExportedFn: func(ctx context.Context, eventIDs []string) error {
			markedIDs = append(markedIDs, eventIDs...)
			return nil
		},
	}

	exporter := NewDataLakeExporter(repo, sinkDir)
	exporter.exportBatch()

	// Verify events were marked
	assert.ElementsMatch(t, []string{"evt-1", "evt-2"}, markedIDs)

	// Verify file was written
	dateStr := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(sinkDir, "token_events_"+dateStr+".jsonl")
	info, err := os.Stat(expectedFile)
	require.NoError(t, err)
	assert.False(t, info.IsDir())

	// Verify JSONL content
	f, err := os.Open(expectedFile)
	require.NoError(t, err)
	defer f.Close()

	var readEvents []domain.TokenEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var evt domain.TokenEvent
		err := json.Unmarshal(scanner.Bytes(), &evt)
		require.NoError(t, err)
		readEvents = append(readEvents, evt)
	}
	require.NoError(t, scanner.Err())

	assert.Len(t, readEvents, 2)
	assert.Equal(t, "evt-1", readEvents[0].EventID)
	assert.Equal(t, "evt-2", readEvents[1].EventID)
}

func TestDataLakeExporter_exportBatch_NoEvents(t *testing.T) {
	sinkDir := t.TempDir()
	markCalled := false
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEventsFn: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return []domain.TokenEvent{}, nil
		},
		markEventsExportedFn: func(ctx context.Context, eventIDs []string) error {
			markCalled = true
			return nil
		},
	}

	exporter := NewDataLakeExporter(repo, sinkDir)
	exporter.exportBatch()

	assert.False(t, markCalled, "MarkEventsExported should not be called if no events are returned")

	// Verify no file was created
	dateStr := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(sinkDir, "token_events_"+dateStr+".jsonl")
	_, err := os.Stat(expectedFile)
	assert.True(t, os.IsNotExist(err), "File should not be created if no events")
}

func TestDataLakeExporter_exportBatch_FetchError(t *testing.T) {
	sinkDir := t.TempDir()
	markCalled := false
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEventsFn: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return nil, errors.New("db error")
		},
		markEventsExportedFn: func(ctx context.Context, eventIDs []string) error {
			markCalled = true
			return nil
		},
	}

	exporter := NewDataLakeExporter(repo, sinkDir)
	exporter.exportBatch()

	assert.False(t, markCalled, "MarkEventsExported should not be called if fetch errors")

	// Verify no file was created
	dateStr := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(sinkDir, "token_events_"+dateStr+".jsonl")
	_, err := os.Stat(expectedFile)
	assert.True(t, os.IsNotExist(err), "File should not be created if fetch errors")
}

func TestDataLakeExporter_exportBatch_MarkError(t *testing.T) {
	sinkDir := t.TempDir()
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEventsFn: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return []domain.TokenEvent{{EventID: "evt-1"}}, nil
		},
		markEventsExportedFn: func(ctx context.Context, eventIDs []string) error {
			return errors.New("db mark error")
		},
	}

	exporter := NewDataLakeExporter(repo, sinkDir)
	exporter.exportBatch()

	// In the current implementation, it logs an error but returns.
	// The file gets written, but events aren't marked as exported in DB.
	dateStr := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(sinkDir, "token_events_"+dateStr+".jsonl")
	_, err := os.Stat(expectedFile)
	assert.NoError(t, err, "File is still created even if mark errors")
}

func TestDataLakeExporter_StartStop(t *testing.T) {
	sinkDir := t.TempDir()
	repo := &mockDatalakeRepo{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		getUnexportedEventsFn: func(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
			return []domain.TokenEvent{}, nil
		},
	}

	exporter := NewDataLakeExporter(repo, sinkDir)

	// Start should spin up a goroutine
	exporter.Start()

	// Yield briefly to allow goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Stop should signal the channel and cause final flush
	exporter.Stop()

	// A small wait to ensure it closes cleanly, mostly avoiding panics
	time.Sleep(10 * time.Millisecond)
}
