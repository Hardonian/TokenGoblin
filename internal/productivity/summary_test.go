package productivity

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func TestBuildSummaryGroupsCostAndDegradedUnknownPricing(t *testing.T) {
	costA := 2.0
	costB := 4.0
	score := 95.0
	events := []domain.TokenEvent{
		{
			TenantID:        "tenant-a",
			EventID:         "evt-a",
			WorkerID:        "worker-a",
			WorkerName:      "Worker A",
			Provider:        "demo",
			ModelID:         "efficient-model",
			TaskCategory:    "classification",
			TotalTokens:     100,
			CostEstimateUSD: &costA,
			LatencyMs:       100,
			OutputStatus:    domain.OutputAccepted,
			ReviewScore:     &score,
		},
		{
			TenantID:        "tenant-a",
			EventID:         "evt-b",
			WorkerID:        "worker-b",
			WorkerName:      "Worker B",
			Provider:        "demo",
			ModelID:         "expensive-model",
			TaskCategory:    "research",
			TotalTokens:     500,
			CostEstimateUSD: &costB,
			LatencyMs:       300,
			OutputStatus:    domain.OutputAccepted,
			ReviewScore:     &score,
		},
		{
			TenantID:       "tenant-a",
			EventID:        "evt-c",
			WorkerID:       "worker-c",
			WorkerName:     "Worker C",
			Provider:       "mystery",
			ModelID:        "unknown",
			TaskCategory:   "research",
			TotalTokens:    200,
			OutputStatus:   domain.OutputFailed,
			CostIsDegraded: true,
		},
	}

	summary := BuildSummary("tenant-a", events, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if summary.TotalEvents != 3 {
		t.Fatalf("expected 3 events, got %d", summary.TotalEvents)
	}
	if summary.TotalCostUSD != 6 {
		t.Fatalf("expected total cost 6, got %.2f", summary.TotalCostUSD)
	}
	if summary.UnknownCostEventCount != 1 {
		t.Fatalf("expected one unknown cost event, got %d", summary.UnknownCostEventCount)
	}
	if summary.CostPerAcceptedOutputWithReview == nil || *summary.CostPerAcceptedOutputWithReview != 3 {
		t.Fatalf("expected cost per accepted reviewed output 3, got %#v", summary.CostPerAcceptedOutputWithReview)
	}
	if len(summary.Degraded) == 0 || summary.Degraded[0].Code != "unknown_pricing_present" {
		t.Fatalf("expected degraded unknown pricing summary, got %#v", summary.Degraded)
	}
}
