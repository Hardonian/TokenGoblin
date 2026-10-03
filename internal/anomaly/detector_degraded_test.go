package anomaly

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_DegradedPaths(t *testing.T) {
	thresholds := DefaultThresholds()
	fixedNow := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	t.Run("Empty Prior Events Slice Does Not Panic Nor Trigger Spikes", func(t *testing.T) {
		event := domain.TokenEvent{
			EventID:      "evt_empty_prior",
			TenantID:     "tenant_1",
			WorkerID:     "worker_1",
			TotalTokens:  1000,
			OutputStatus: domain.OutputSucceeded,
		}

		signals := Detect(event, nil, fixedNow, thresholds)
		assert.Empty(t, signals)

		signals = Detect(event, []domain.TokenEvent{}, fixedNow, thresholds)
		assert.Empty(t, signals)
	})

	t.Run("Nil Cost Estimate Does Not Trigger Spend Spike", func(t *testing.T) {
		event := domain.TokenEvent{
			EventID:         "evt_nil_cost",
			TenantID:        "tenant_1",
			WorkerID:        "worker_1",
			CostEstimateUSD: nil,
			OutputStatus:    domain.OutputSucceeded,
		}
		cost := 0.05
		prior := []domain.TokenEvent{
			{CostEstimateUSD: &cost, WorkerID: "worker_1"},
			{CostEstimateUSD: &cost, WorkerID: "worker_1"},
			{CostEstimateUSD: &cost, WorkerID: "worker_1"},
		}

		signals := Detect(event, prior, fixedNow, thresholds)
		assert.Empty(t, signals)
	})

	t.Run("Cost Degraded With Unknown Model Pricing", func(t *testing.T) {
		event := domain.TokenEvent{
			EventID:          "evt_degraded_pricing",
			TenantID:         "tenant_1",
			WorkerID:         "worker_1",
			CostIsDegraded:   true,
			CostDegradedCode: "unknown_model_pricing",
			OutputStatus:     domain.OutputSucceeded,
		}

		signals := Detect(event, nil, fixedNow, thresholds)
		require.Len(t, signals, 1)
		assert.Equal(t, domain.AnomalyUnknownModelPricing, signals[0].Type)
		assert.Equal(t, domain.SeverityMed, signals[0].Severity)
		assert.Equal(t, fixedNow, signals[0].DetectedAt)
	})

	t.Run("Zero Timestamp Falls Back To Now", func(t *testing.T) {
		event := domain.TokenEvent{
			EventID:          "evt_zero_time",
			CostIsDegraded:   true,
			CostDegradedCode: "unknown_model_pricing",
			Timestamp:        time.Time{},
		}

		signals := Detect(event, nil, fixedNow, thresholds)
		require.Len(t, signals, 1)
		assert.Equal(t, fixedNow, signals[0].DetectedAt)
	})

	t.Run("Prior Events With Nil Cost Estimates Handle Gracefully", func(t *testing.T) {
		eventCost := 10.0
		event := domain.TokenEvent{
			EventID:         "evt_event_cost",
			CostEstimateUSD: &eventCost,
			OutputStatus:    domain.OutputSucceeded,
		}
		prior := []domain.TokenEvent{
			{CostEstimateUSD: nil},
			{CostEstimateUSD: nil},
			{CostEstimateUSD: nil},
		}

		// Since costCount is 0 (< 3), no spend spike should trigger
		signals := Detect(event, prior, fixedNow, thresholds)
		assert.Empty(t, signals)
	})

	t.Run("Extreme Latency and Zero Latency Handled Without Panic", func(t *testing.T) {
		event := domain.TokenEvent{
			EventID:      "evt_lat",
			LatencyMs:    0,
			OutputStatus: domain.OutputSucceeded,
		}
		prior := []domain.TokenEvent{
			{LatencyMs: 50},
			{LatencyMs: 60},
			{LatencyMs: 70},
		}

		signals := Detect(event, prior, fixedNow, thresholds)
		assert.Empty(t, signals)
	})
}
