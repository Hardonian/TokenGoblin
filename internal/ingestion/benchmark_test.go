package ingestion

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func BenchmarkIngestTokenEventBatch(b *testing.B) {
	ctx := context.Background()
	repo, err := storage.OpenSQLite(ctx, ":memory:")
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()

	registry := cost.LoadRegistry(ctx, cost.RegistryConfig{DisableDefaults: true})
	service := NewService(repo, registry)

	// Create tenant and initial event
	err = repo.UpsertTenant(ctx, domain.Tenant{
		TenantID:      "tenant-a",
		Name:          "tenant-a",
		UsageLimitUSD: 1000.0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	if err != nil {
		b.Fatalf("upsert tenant: %v", err)
	}

	batchSize := 100
	events := make([]domain.TokenEvent, batchSize)
	for i := 0; i < batchSize; i++ {
		events[i] = domain.TokenEvent{
			EventID:      fmt.Sprintf("evt-%d", i),
			TenantID:     "tenant-a",
			WorkerID:     "worker-a",
			Provider:     "demo",
			ModelID:      "model-a",
			PromptTokens: 10,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just ingest the batch, do not wait for workers
		// We use varying event IDs so we don't have unique constraint issues if workers were processing
		// But workers aren't started, so it just queues them up or returns buffer full.
		// Wait, if we run it b.N times, the buffer of 1000 will fill up!
		// We need to drain the buffer or increase size.
		// We can just drain it in a goroutine.
		go func() {
			for range service.eventQueue {
			}
		}()
		_, err := service.IngestTokenEventBatch(ctx, "tenant-a", events)
		if err != nil {
			b.Fatalf("ingest batch: %v", err)
		}
	}
}
