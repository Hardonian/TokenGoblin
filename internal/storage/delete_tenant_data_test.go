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
		b.Fatalf("failed to open sqlite: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Establish a baseline by running the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup test data for the iteration
		tenantID := fmt.Sprintf("test-tenant-%d", i)

		// Setup a bunch of data that will be deleted
		repo.UpsertTenant(ctx, domain.Tenant{TenantID: tenantID, Name: "Test Tenant"})

		for j := 0; j < 100; j++ {
			repo.SaveTokenEvent(ctx, domain.TokenEvent{
				TenantID:     tenantID,
				EventID:      fmt.Sprintf("event-%d-%d", i, j),
				Provider:     "test",
				ModelID:      "test-model",
				WorkerID:     "worker1",
				WorkerName:   "Worker 1",
				InputTokens:  10,
				OutputTokens: 10,
				Timestamp:    time.Now(),
			})
		}

		b.StartTimer()
		err := repo.DeleteTenantData(ctx, tenantID)
		if err != nil {
			b.Fatalf("failed to delete tenant data: %v", err)
		}
	}
}
