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
	"github.com/Hardonian/TokenGoblin/internal/storage/clickhouse"
)

func main() {
	ctx := context.Background()
	repo, err := storage.OpenFromEnv(ctx)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// Clock base must match the demo event timeline (see demo.DefaultBaseTime).
	base := demo.DefaultBaseTime()
	service := ingestion.NewService(repo, cost.LoadRegistry(ctx, cost.ConfigFromEnv())).WithClock(func() time.Time {
		return base
	})

	// Optional ClickHouse telemetry mirror. If TG_CLICKHOUSE_ADDR is set, a
	// broken mirror is a hard failure here (misconfigured seed must not pass
	// silently) and the mirrored row count is verified before claiming success.
	sink, sinkErr := clickhouse.MaybeEventSinkFromEnv(ctx)
	if sinkErr != nil {
		log.Fatalf("clickhouse telemetry configured but unavailable: %v", sinkErr)
	}
	if sink != nil {
		defer func() { _ = sink.Close() }()
		service.WithTelemetrySink(sink)
	}

	service.StartWorker(ctx)

	tenantID := os.Getenv("TG_DEMO_TENANT_ID")
	if tenantID == "" {
		tenantID = demo.DefaultTenantID
	}
	if err := demo.Seed(ctx, repo, service, tenantID); err != nil {
		log.Fatalf("seed demo: %v", err)
	}
	events := demo.Events(tenantID)
	if sink != nil {
		mirrored, err := sink.CountTokenEvents(ctx, tenantID)
		if err != nil {
			log.Fatalf("verify clickhouse mirror: %v", err)
		}
		if mirrored != int64(len(events)) {
			log.Fatalf("clickhouse mirror incomplete: %d of %d events mirrored", mirrored, len(events))
		}
		// #nosec G706 -- CLI demo seed utility
		log.Printf("seeded demo tenant %q: %d usage events generated AND verified persisted (clickhouse mirror: %d)", tenantID, len(events), mirrored)
		return
	}
	// #nosec G706 -- CLI demo seed utility
	log.Printf("seeded demo tenant %q: %d usage events generated AND verified persisted", tenantID, len(events))
}
