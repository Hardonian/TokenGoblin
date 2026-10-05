package demo

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// slowRepo delays every event write so asynchronous ingestion cannot possibly
// finish inside the old fixed 300ms sleep this suite guards against (23 events
// x 25ms > 300ms). If Seed ever regresses to a blind sleep, these tests fail.
type slowRepo struct {
	storage.Repository
	delay time.Duration
}

func (r *slowRepo) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	time.Sleep(r.delay)
	return r.Repository.SaveTokenEvent(ctx, event)
}

// failingRepo drops every event write after the worker's retries are exhausted.
type failingRepo struct {
	storage.Repository
}

func (r *failingRepo) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	return errors.New("write refused")
}

func TestSeedWaitsForAsyncIngestionAndVerifiesPersistence(t *testing.T) {
	ctx := context.Background()
	base, err := storage.OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = base.Close() }()

	repo := &slowRepo{Repository: base, delay: 25 * time.Millisecond}
	service := ingestion.NewService(repo, cost.LoadRegistry(ctx, cost.RegistryConfig{}))
	service.StartWorker(ctx)

	tenantID := "seed-verify"
	if err := Seed(ctx, repo, service, tenantID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	events := Events(tenantID)
	persisted, err := base.ListTokenEvents(ctx, tenantID, len(events)+1)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(persisted) != len(events) {
		t.Fatalf("seed returned success but only %d of %d events persisted", len(persisted), len(events))
	}
}

type mirrorSink struct {
	cleared int
	mirrors int
}

func (m *mirrorSink) SinkTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	m.mirrors++
	return nil
}

func (m *mirrorSink) SinkAnomalySignals(ctx context.Context, signals []domain.AnomalySignal) error {
	return nil
}

func (m *mirrorSink) DeleteTenantEvents(ctx context.Context, tenantID string) error {
	m.cleared++
	m.mirrors = 0
	return nil
}

func TestSeedResetsTelemetryMirror(t *testing.T) {
	ctx := context.Background()
	base, err := storage.OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = base.Close() }()

	sink := &mirrorSink{}
	service := ingestion.NewService(base, cost.LoadRegistry(ctx, cost.RegistryConfig{})).WithTelemetrySink(sink)
	service.StartWorker(ctx)

	// Re-seeding must never accumulate mirrored rows (a reseed previously
	// doubled analytics counts because only the primary store was reset).
	for i := 0; i < 2; i++ {
		if err := Seed(ctx, base, service, "seed-mirror"); err != nil {
			t.Fatalf("seed pass %d: %v", i+1, err)
		}
	}
	if sink.cleared != 2 {
		t.Fatalf("mirror should be cleared once per seed, got %d clears", sink.cleared)
	}
	if sink.mirrors != len(Events("seed-mirror")) {
		t.Fatalf("mirror holds %d events after 2 seeds, want exactly one seed's worth (%d)",
			sink.mirrors, len(Events("seed-mirror")))
	}
}

func TestSeedFailsWhenEventsDoNotPersist(t *testing.T) {
	ctx := context.Background()
	base, err := storage.OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = base.Close() }()

	repo := &failingRepo{Repository: base}
	service := ingestion.NewService(repo, cost.LoadRegistry(ctx, cost.RegistryConfig{}))
	service.StartWorker(ctx)

	// Seed MUST report failure instead of claiming a seeded count that never
	// landed in storage.
	if err := Seed(ctx, repo, service, "seed-fail"); err == nil {
		t.Fatal("expected seed to fail when no events persist, got nil")
	}
}
