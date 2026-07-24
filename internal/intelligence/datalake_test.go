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

type mockRepoForDataLake struct {
	*storage.UnavailableRepository
	eventsToReturn []domain.TokenEvent
	fetchError     error
	exportedIDs    []string
	exportError    error
}

func (m *mockRepoForDataLake) GetUnexportedEvents(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	if len(m.eventsToReturn) > limit {
		return m.eventsToReturn[:limit], nil
	}
	return m.eventsToReturn, nil
}

func (m *mockRepoForDataLake) MarkEventsExported(ctx context.Context, eventIDs []string) error {
	if m.exportError != nil {
		return m.exportError
	}
	m.exportedIDs = append(m.exportedIDs, eventIDs...)
	return nil
}

func TestNewDataLakeExporter_DefaultDir(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()

	// We want to test the default logic which uses "./data/datalake".
	// To avoid polluting the real directory, we change the working directory for the test.
	origWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Chdir(origWd)
	})

	repo := &mockRepoForDataLake{UnavailableRepository: storage.NewUnavailableRepository(nil)}

	exporter := NewDataLakeExporter(repo, "")
	assert.Equal(t, "./data/datalake", exporter.sinkDir)

	info, err := os.Stat("./data/datalake")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDataLakeExporter_exportBatch(t *testing.T) {
	tmpDir := t.TempDir()

	now := time.Now().UTC()
	events := []domain.TokenEvent{
		{EventID: "evt-1", TenantID: "t-1", ModelID: "gpt-4", OutputTokens: 10, Timestamp: now},
		{EventID: "evt-2", TenantID: "t-1", ModelID: "gpt-4", OutputTokens: 20, Timestamp: now},
	}

	repo := &mockRepoForDataLake{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		eventsToReturn:        events,
	}

	exporter := NewDataLakeExporter(repo, tmpDir)

	// Execute
	exporter.exportBatch()

	// Verify repo interactions
	assert.ElementsMatch(t, []string{"evt-1", "evt-2"}, repo.exportedIDs)

	// Verify file was written
	dateStr := time.Now().Format("2006-01-02")
	expectedFilename := filepath.Join(tmpDir, "token_events_"+dateStr+".jsonl")

	info, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.True(t, info.Size() > 0)

	// Verify file content
	f, err := os.Open(expectedFilename)
	require.NoError(t, err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var readEvents []domain.TokenEvent
	for scanner.Scan() {
		var evt domain.TokenEvent
		err := json.Unmarshal(scanner.Bytes(), &evt)
		require.NoError(t, err)
		readEvents = append(readEvents, evt)
	}

	assert.Len(t, readEvents, 2)
	assert.Equal(t, "evt-1", readEvents[0].EventID)
	assert.Equal(t, "evt-2", readEvents[1].EventID)
}

func TestDataLakeExporter_exportBatch_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	repo := &mockRepoForDataLake{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		eventsToReturn:        []domain.TokenEvent{},
	}

	exporter := NewDataLakeExporter(repo, tmpDir)
	exporter.exportBatch()

	assert.Empty(t, repo.exportedIDs)

	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "should not create any files when there are no events")
}

func TestDataLakeExporter_exportBatch_FetchError(t *testing.T) {
	tmpDir := t.TempDir()

	repo := &mockRepoForDataLake{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		fetchError:            errors.New("db down"),
	}

	exporter := NewDataLakeExporter(repo, tmpDir)
	exporter.exportBatch()

	assert.Empty(t, repo.exportedIDs)

	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDataLakeExporter_StartStop(t *testing.T) {
	tmpDir := t.TempDir()

	now := time.Now().UTC()
	events := []domain.TokenEvent{
		{EventID: "evt-1", TenantID: "t-1", ModelID: "gpt-4", OutputTokens: 10, Timestamp: now},
	}

	repo := &mockRepoForDataLake{
		UnavailableRepository: storage.NewUnavailableRepository(nil),
		eventsToReturn:        events,
	}

	exporter := NewDataLakeExporter(repo, tmpDir)
	// Override interval so it ticks quickly
	exporter.interval = 10 * time.Millisecond

	exporter.Start()

	// Wait a bit for the ticker to fire
	time.Sleep(50 * time.Millisecond)

	exporter.Stop()

	// Wait a bit for the goroutine to finish and do the final flush
	time.Sleep(10 * time.Millisecond)

	// Since the batch limit wasn't restricted and the mock always returns the same events,
	// it might have exported multiple times. We just check it exported *something*.
	assert.NotEmpty(t, repo.exportedIDs)
}
