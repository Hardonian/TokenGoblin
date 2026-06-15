package storage_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func BenchmarkDeleteTenantData(b *testing.B) {
	// Create an in-memory SQLite repository
	repo, err := storage.OpenSQLite(context.Background(), ":memory:")
	if err != nil {
		b.Fatalf("failed to open sqlite: %w", err)
	}
	defer func() { _ = repo.Close() }()

	ctx := context.Background()

	// Establish a baseline by running the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup test data for the iteration
		tenantID := fmt.Sprintf("test-tenant-%d", i)

		// Setup a bunch of data that will be deleted
		if err := repo.UpsertTenant(ctx, domain.Tenant{TenantID: tenantID, Name: "Test Tenant"}); err != nil {
			b.Fatalf("failed to upsert tenant: %w", err)
		}

		for j := 0; j < 100; j++ {
			if err := repo.SaveTokenEvent(ctx, domain.TokenEvent{
				TenantID:     tenantID,
				EventID:      fmt.Sprintf("event-%d-%d", i, j),
				Provider:     "test",
				ModelID:      "test-model",
				WorkerID:     "worker1",
				WorkerName:   "Worker 1",
				InputTokens:  10,
				OutputTokens: 10,
				Timestamp:    time.Now(),
			}); err != nil {
				b.Fatalf("failed to save token event: %w", err)
			}
		}

		b.StartTimer()
		err := repo.DeleteTenantData(ctx, tenantID)
		if err != nil {
			b.Fatalf("failed to delete tenant data: %w", err)
		}
	}
}
