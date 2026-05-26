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
	for _, event := range Events(tenantID) {
		if _, err := service.IngestTokenEvent(ctx, tenantID, event); err != nil {
			return fmt.Errorf("seed %s: %w", event.EventID, err)
		}
	}
	return nil
}

func Events(tenantID string) []domain.TokenEvent {
	base := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
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
			Tags:             map[string]string{"demo": "true", "profile": "efficient"},
		})
	}

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
			Tags:             map[string]string{"demo": "true", "profile": "expensive"},
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
			Tags:             map[string]string{"demo": "true", "profile": "spike"},
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
			Tags:             map[string]string{"demo": "true", "profile": "expensive"},
		},
	)

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
			Tags:             map[string]string{"demo": "true", "profile": "unknown-pricing"},
		})
	}

	return events
}

func ptr(value float64) *float64 {
	return &value
}
