package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/demo"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func main() {
	ctx := context.Background()
	repo, err := storage.OpenFromEnv(ctx)
	if err != nil {
		log.Fatalf("open storage: %w", err)
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
		log.Fatalf("seed demo: %w", err)
	}

	var summary domain.ProductivitySummary
	var overview = func() (int, error) {
		s, err := service.Overview(ctx, tenantID)
		if err != nil {
			return 0, err
		}
		summary = s
		return s.TotalEvents, nil
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		count, err := overview()
		if err != nil {
			log.Fatalf("overview: %w", err)
		}
		if count >= 20 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if summary.TotalEvents < 20 {
		log.Fatalf("expected at least 20 events, got %d", summary.TotalEvents)
	}
	if len(summary.CostByWorker) != 3 {
		log.Fatalf("expected 3 workers, got %d", len(summary.CostByWorker))
	}
	if summary.AnomalyCount < 3 {
		log.Fatalf("expected at least 3 anomalies, got %d", summary.AnomalyCount)
	}
	if summary.UnknownCostEventCount == 0 {
		log.Fatal("expected unknown pricing degraded events")
	}
	foundExpensiveWorker := false
	for _, driver := range summary.TopCostDrivers {
		if driver.Type == "worker" && driver.Key == "worker-expensive" {
			foundExpensiveWorker = true
			break
		}
	}
	if !foundExpensiveWorker {
		log.Fatalf("expected worker-expensive in top cost drivers, got %#v", summary.TopCostDrivers)
	}

	fmt.Printf("smoke ok: tenant=%s events=%d workers=%d anomalies=%d unknown_cost_events=%d\n",
		tenantID, summary.TotalEvents, len(summary.CostByWorker), summary.AnomalyCount, summary.UnknownCostEventCount)
}
