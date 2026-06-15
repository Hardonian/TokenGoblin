package ingestion

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func TestIngestTokenEventComputesInternalCostAndStoresExternalEstimate(t *testing.T) {
	ctx := context.Background()
	service, repo := testService(t)
	defer func() { _ = repo.Close() }()

	clientCost := 999.0
	result, err := service.IngestTokenEvent(ctx, "tenant-a", domain.TokenEvent{
		EventID:          "evt-1",
		TenantID:         "tenant-a",
		Timestamp:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		WorkerID:         "worker-a",
		Provider:         "demo",
		ModelID:          "efficient-model",
		PromptTokens:     1_000,
		CompletionTokens: 500,
		CostEstimateUSD:  &clientCost,
		ExternalEstimate: &domain.ExternalEstimate{CostUSD: 1.23},
		OutputStatus:     domain.OutputAccepted,
		ReviewScore:      ptr(90),
	})
	if err != nil {
		t.Fatalf("ingest: %w", err)
	}
	if result.Event.CostEstimateUSD == nil {
		t.Fatal("expected internal cost")
	}
	if *result.Event.CostEstimateUSD == clientCost {
		t.Fatal("client-supplied cost was trusted")
	}
	if len(result.Warnings) == 0 || result.Warnings[0].Code != "ignored_client_cost" {
		t.Fatalf("expected ignored client cost warning, got %#v", result.Warnings)
	}
	waitForEventCount(t, repo, "tenant-a", 1)
	events, err := repo.ListTokenEvents(ctx, "tenant-a", 10)
	if err != nil {
		t.Fatalf("list events: %w", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].ExternalEstimate == nil || events[0].ExternalEstimate.Currency != "USD" {
		t.Fatalf("expected external estimate persisted with USD default, got %#v", events[0].ExternalEstimate)
	}
}

func TestIngestRejectsCrossTenantPayload(t *testing.T) {
	ctx := context.Background()
	service, repo := testService(t)
	defer func() { _ = repo.Close() }()

	_, err := service.IngestTokenEvent(ctx, "tenant-a", domain.TokenEvent{
		EventID:      "evt-tenant-mismatch",
		TenantID:     "tenant-b",
		WorkerID:     "worker-a",
		Provider:     "demo",
		ModelID:      "efficient-model",
		PromptTokens: 1,
	})
	var mismatch TenantMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected tenant mismatch, got %T %v", err, err)
	}
}

func TestIngestUnknownPricingCreatesDegradedAnomaly(t *testing.T) {
	ctx := context.Background()
	service, repo := testService(t)
	defer func() { _ = repo.Close() }()

	result, err := service.IngestTokenEvent(ctx, "tenant-a", domain.TokenEvent{
		EventID:          "evt-unknown",
		Timestamp:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		WorkerID:         "worker-a",
		Provider:         "mystery",
		ModelID:          "unknown",
		PromptTokens:     100,
		CompletionTokens: 50,
	})
	if err != nil {
		t.Fatalf("ingest: %w", err)
	}
	if len(result.Degraded) == 0 || result.Degraded[0].Code != "unknown_model_pricing" {
		t.Fatalf("expected unknown pricing degradation, got %#v", result.Degraded)
	}
	waitForAnomalyCount(t, repo, "tenant-a", 1)
	anomalies, err := repo.ListAnomalySignals(ctx, "tenant-a", 10)
	if err != nil {
		t.Fatalf("list anomalies: %w", err)
	}
	if len(anomalies) != 1 || anomalies[0].Type != domain.AnomalyUnknownModelPricing {
		t.Fatalf("expected unknown pricing anomaly, got %#v", anomalies)
	}
}

func TestIngestPersistsOutputAnalysis(t *testing.T) {
	ctx := context.Background()
	service, repo := testService(t)
	defer func() { _ = repo.Close() }()

	_, err := service.IngestTokenEvent(ctx, "tenant-a", domain.TokenEvent{
		EventID:       "evt-analysis",
		WorkerID:      "worker-a",
		Provider:      "demo",
		ModelID:       "efficient-model",
		InputTokens:   100,
		OutputTokens:  300,
		PromptExcerpt: "Write an answer.",
		OutputExcerpt: "This is a long unstructured answer without evidence markers that repeats. This is a long unstructured answer without evidence markers that repeats.",
		OutputStatus:  domain.OutputAccepted,
	})
	if err != nil {
		t.Fatalf("ingest: %w", err)
	}
	waitForAnalysisCount(t, repo, "tenant-a", 1)
	analyses, err := repo.ListOutputAnalyses(ctx, "tenant-a", 10)
	if err != nil {
		t.Fatalf("list analyses: %w", err)
	}
	if len(analyses) != 1 {
		t.Fatalf("expected one analysis, got %d", len(analyses))
	}
	if analyses[0].EfficiencyScore >= 100 || len(analyses[0].Issues) == 0 {
		t.Fatalf("expected scored analysis with issues, got %#v", analyses[0])
	}
}

func TestIngestTokenEventReusesQueuedCostCalculation(t *testing.T) {
	ctx := context.Background()
	baseRepo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %w", err)
	}
	defer func() { _ = baseRepo.Close() }()

	repo := &pricingCountRepo{Repository: baseRepo}
	service := NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{})).WithClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	service.StartWorker(context.Background())

	_, err = service.IngestTokenEvent(ctx, "tenant-a", domain.TokenEvent{
		EventID:      "evt-priced-once",
		WorkerID:     "worker-a",
		Provider:     "demo",
		ModelID:      "efficient-model",
		InputTokens:  100,
		OutputTokens: 50,
	})
	if err != nil {
		t.Fatalf("ingest: %w", err)
	}
	waitForEventCount(t, repo, "tenant-a", 1)

	if got := repo.pricingLookups.Load(); got != 1 {
		t.Fatalf("expected one pricing lookup, got %d", got)
	}
}

func testService(t *testing.T) (*ExecutionService, storage.Repository) {
	t.Helper()
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %w", err)
	}
	service := NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{})).WithClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	service.StartWorker(context.Background())
	return service, repo
}

func ptr(value float64) *float64 {
	return &value
}

type pricingCountRepo struct {
	storage.Repository
	pricingLookups atomic.Int64
}

func (r *pricingCountRepo) GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error) {
	r.pricingLookups.Add(1)
	return r.Repository.GetPricingOverride(ctx, tenantID, provider, modelID)
}

func waitForEventCount(t *testing.T, repo storage.Repository, tenantID string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		events, err := repo.ListTokenEvents(context.Background(), tenantID, want)
		if err == nil && len(events) >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d persisted events", want)
}

func waitForAnomalyCount(t *testing.T, repo storage.Repository, tenantID string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		anomalies, err := repo.ListAnomalySignals(context.Background(), tenantID, want)
		if err == nil && len(anomalies) >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d anomaly signals", want)
}

func waitForAnalysisCount(t *testing.T, repo storage.Repository, tenantID string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		analyses, err := repo.ListOutputAnalyses(context.Background(), tenantID, want)
		if err == nil && len(analyses) >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d output analyses", want)
}
