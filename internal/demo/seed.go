package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

const DefaultTenantID = "demo-tenant"

func Seed(ctx context.Context, repo storage.Repository, service ingestion.Service, tenantID string) error {
	if tenantID == "" {
		tenantID = DefaultTenantID
	}
	if err := repo.DeleteTenantData(ctx, tenantID); err != nil {
		return err
	}
	// DEMO-ONLY provisioning exception: for PRODUCTION tenants the billing
	// lifecycle is the ONLY writer of `tier` (see internal/billing). The
	// synthetic demo tenant is provisioned on enterprise so seeded demos can
	// exercise the tier-gated analytics surfaces (e.g. zombie agents) end to
	// end. Do not copy this pattern into product code paths.
	if err := repo.UpsertTenant(ctx, domain.Tenant{
		TenantID:      tenantID,
		Name:          "Demo Tenant",
		Tier:          "enterprise",
		UsageLimitUSD: 10,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("seed %s: provision demo tenant: %w", tenantID, err)
	}
	// Reset the telemetry mirror alongside the primary store so re-seeding can
	// never leave (or double-count) rows the source of truth no longer has.
	if resetter, ok := service.(interface {
		ClearTelemetryMirror(ctx context.Context, tenantID string) error
	}); ok {
		if err := resetter.ClearTelemetryMirror(ctx, tenantID); err != nil {
			return fmt.Errorf("seed %s: reset telemetry mirror: %w", tenantID, err)
		}
	}
	events := Events(tenantID)
	for _, event := range events {
		if _, err := service.IngestTokenEvent(ctx, tenantID, event); err != nil {
			return fmt.Errorf("seed %s: %w", event.EventID, err)
		}
	}
	// Ingestion is asynchronous (buffered queue + background worker). Wait for
	// the queue to drain, then VERIFY what actually landed in storage — never
	// claim a seeded count from what was merely generated.
	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := service.WaitIdle(waitCtx); err != nil {
		return fmt.Errorf("seed %s: %w", tenantID, err)
	}
	persisted, err := repo.ListTokenEvents(ctx, tenantID, len(events)+1)
	if err != nil {
		return fmt.Errorf("seed %s: verify persisted events: %w", tenantID, err)
	}
	if len(persisted) != len(events) {
		return fmt.Errorf("seed %s incomplete: persisted %d of %d events", tenantID, len(persisted), len(events))
	}
	return nil
}

// DefaultBaseTime anchors the demo event timeline one hour in the past so the
// data behaves like live data. A fixed historical date made seeded events look
// like ancient telemetry: any store with retention (e.g. ClickHouse
// token_events, 90-day TTL) deletes them at merge time, and cost rollups land
// in a closed billing month. The relative offsets between events are kept
// identical, so structural assertions stay deterministic.
func DefaultBaseTime() time.Time {
	return time.Now().UTC().Truncate(time.Minute).Add(-time.Hour)
}

func Events(tenantID string) []domain.TokenEvent {
	base := DefaultBaseTime()
	var events []domain.TokenEvent

	events = append(events, efficientEvents(tenantID, base)...)
	events = append(events, expensiveEvents(tenantID, base)...)
	events = append(events, unknownEvents(tenantID, base)...)

	return events
}

func efficientEvents(tenantID string, base time.Time) []domain.TokenEvent {
	var events []domain.TokenEvent
	for i := 0; i < 9; i++ {
		events = append(events, domain.TokenEvent{
			EventID:          fmt.Sprintf("demo-efficient-%02d", i+1),
			TenantID:         tenantID,
			Timestamp:        base.Add(time.Duration(i) * time.Minute),
			WorkerID:         "worker-efficient",
			WorkerName:       "Efficient Worker",
			JobID:            "job-efficient",
			SessionID:        "session-efficient",
			RunID:            fmt.Sprintf("run-efficient-%02d", i+1),
			Provider:         "demo",
			ModelID:          "efficient-model",
			PromptTokens:     900 + i*20,
			CompletionTokens: 180 + i*5,
			CachedTokens:     120,
			LatencyMs:        650 + i*10,
			TaskCategory:     "classification",
			OutputStatus:     domain.OutputAccepted,
			ReviewScore:      ptr(94),
			TagsJSON:         []byte(`{"demo": "true", "profile": "efficient"}`),
		})
	}
	return events
}

func expensiveEvents(tenantID string, base time.Time) []domain.TokenEvent {
	var events []domain.TokenEvent
	for i := 0; i < 6; i++ {
		events = append(events, domain.TokenEvent{
			EventID:          fmt.Sprintf("demo-expensive-base-%02d", i+1),
			TenantID:         tenantID,
			Timestamp:        base.Add(time.Duration(20+i) * time.Minute),
			WorkerID:         "worker-expensive",
			WorkerName:       "Expensive Worker",
			JobID:            "job-expensive",
			SessionID:        "session-expensive",
			RunID:            fmt.Sprintf("run-expensive-%02d", i+1),
			Provider:         "demo",
			ModelID:          "expensive-model",
			PromptTokens:     5_000,
			CompletionTokens: 1_000,
			LatencyMs:        2_200 + i*50,
			TaskCategory:     "research",
			OutputStatus:     domain.OutputAccepted,
			ReviewScore:      ptr(87),
			TagsJSON:         []byte(`{"demo": "true", "profile": "expensive"}`),
		})
	}

	events = append(events,
		domain.TokenEvent{
			EventID:          "demo-expensive-spike-01",
			TenantID:         tenantID,
			Timestamp:        base.Add(30 * time.Minute),
			WorkerID:         "worker-expensive",
			WorkerName:       "Expensive Worker",
			JobID:            "job-expensive",
			SessionID:        "session-expensive",
			RunID:            "run-expensive-spike",
			Provider:         "demo",
			ModelID:          "expensive-model",
			PromptTokens:     300_000,
			CompletionTokens: 120_000,
			LatencyMs:        45_000,
			TaskCategory:     "research",
			OutputStatus:     domain.OutputAccepted,
			ReviewScore:      ptr(82),
			TagsJSON:         []byte(`{"demo": "true", "profile": "spike"}`),
		},
		domain.TokenEvent{
			EventID:          "demo-expensive-retry-01",
			TenantID:         tenantID,
			Timestamp:        base.Add(31 * time.Minute),
			WorkerID:         "worker-expensive",
			WorkerName:       "Expensive Worker",
			JobID:            "job-expensive",
			SessionID:        "session-expensive",
			RunID:            "run-expensive-retry",
			Provider:         "demo",
			ModelID:          "expensive-model",
			PromptTokens:     9_000,
			CompletionTokens: 1_500,
			LatencyMs:        2_800,
			TaskCategory:     "research",
			OutputStatus:     domain.OutputRejected,
			ReviewScore:      ptr(41),
			TagsJSON:         []byte(`{"demo": "true", "profile": "expensive"}`),
		},
	)
	return events
}

func unknownEvents(tenantID string, base time.Time) []domain.TokenEvent {
	var events []domain.TokenEvent
	for i := 0; i < 6; i++ {
		status := domain.OutputFailed
		if i > 2 {
			status = domain.OutputPending
		}
		events = append(events, domain.TokenEvent{
			EventID:          fmt.Sprintf("demo-unknown-%02d", i+1),
			TenantID:         tenantID,
			Timestamp:        base.Add(time.Duration(45+i) * time.Minute),
			WorkerID:         "worker-unknown",
			WorkerName:       "Unknown Pricing Worker",
			JobID:            "job-unknown",
			SessionID:        "session-unknown",
			RunID:            fmt.Sprintf("run-unknown-%02d", i+1),
			Provider:         "mysteryai",
			ModelID:          "unknown-v9",
			PromptTokens:     2_400 + i*100,
			CompletionTokens: 700 + i*30,
			LatencyMs:        1_900 + i*20,
			TaskCategory:     "summarization",
			OutputStatus:     status,
			ReviewScore:      nil,
			TagsJSON:         []byte(`{"demo": "true", "profile": "unknown-pricing"}`),
		})
	}
	return events
}

func ptr(value float64) *float64 {
	return &value
}
