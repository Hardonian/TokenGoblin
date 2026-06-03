package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/demo"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func main() {
	ctx := context.Background()
	repo, err := storage.OpenFromEnv(ctx)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer func() { _ = repo.Close() }()

	base := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	service := ingestion.NewService(repo, cost.LoadRegistry(ctx, cost.ConfigFromEnv())).WithClock(func() time.Time {
		return base
	})
	service.StartWorker(ctx)

	tenantID := os.Getenv("TG_DEMO_TENANT_ID")
	if tenantID == "" {
		tenantID = demo.DefaultTenantID
	}
	if err := demo.Seed(ctx, repo, service, tenantID); err != nil {
		log.Fatalf("seed demo: %v", err)
	}
	log.Printf("seeded demo tenant %q with %d usage events", tenantID, len(demo.Events(tenantID)))
}
