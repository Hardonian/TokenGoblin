package ingestion

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type recordingSink struct {
	mu       sync.Mutex
	events   []domain.TokenEvent
	err      error
	cleared  []string
}

func (r *recordingSink) SinkTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.events = append(r.events, event)
	return nil
}

func (r *recordingSink) DeleteTenantEvents(ctx context.Context, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleared = append(r.cleared, tenantID)
	r.events = nil
	return nil
}

func (r *recordingSink) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func (r *recordingSink) clearedTenants() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.cleared...)
}

func TestTelemetrySinkReceivesPersistedEvents(t *testing.T) {
	ctx := context.Background()
	repo, err := storage.OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	sink := &recordingSink{}
	service := NewService(repo, cost.LoadRegistry(ctx, cost.RegistryConfig{})).WithTelemetrySink(sink)
	service.StartWorker(ctx)

	for _, id := range []string{"evt-sink-1", "evt-sink-2"} {
		if _, err := service.IngestTokenEvent(ctx, "tenant-sink", domain.TokenEvent{
			EventID:  id,
			WorkerID: "worker-a",
			ModelID:  "model-a",
			Provider: "demo",
		}); err != nil {
			t.Fatalf("ingest %s: %v", id, err)
		}
	}
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("wait idle: %v", err)
	}

	if got := sink.count(); got != 2 {
		t.Fatalf("sink received %d events, want 2", got)
	}
}

func TestTelemetrySinkFailureNeverFailsPrimaryWrites(t *testing.T) {
	ctx := context.Background()
	repo, err := storage.OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// A sink that always fails must not block, fail, or re-route the primary
	// persistence — it is a secondary mirror only.
	sink := &recordingSink{err: errors.New("cluster down")}
	service := NewService(repo, cost.LoadRegistry(ctx, cost.RegistryConfig{})).WithTelemetrySink(sink)
	service.StartWorker(ctx)

	if _, err := service.IngestTokenEvent(ctx, "tenant-sink-fail", domain.TokenEvent{
		EventID:  "evt-sink-fail",
		WorkerID: "worker-a",
		ModelID:  "model-a",
		Provider: "demo",
	}); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("wait idle: %v", err)
	}

	events, err := repo.ListTokenEvents(ctx, "tenant-sink-fail", 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("primary store should hold 1 event despite sink failure, got %d", len(events))
	}
	if got := sink.count(); got != 0 {
		t.Fatalf("failed sink should have recorded 0 events, got %d", got)
	}
}