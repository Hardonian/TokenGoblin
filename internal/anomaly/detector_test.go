package anomaly

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
)

func floatPtr(f float64) *float64 {
	return &f
}

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	assert.Equal(t, float64(3), thresholds.SpendSpikeMultiplier)
	assert.Equal(t, float64(1), thresholds.SpendSpikeMinimumUSD)
	assert.Equal(t, float64(3), thresholds.TokenSpikeMultiplier)
	assert.Equal(t, float64(10000), thresholds.TokenSpikeMinimumTokens)
	assert.Equal(t, float64(3), thresholds.LatencySpikeMultiplier)
	assert.Equal(t, float64(10000), thresholds.LatencySpikeMinimumMs)
	assert.Equal(t, 10, thresholds.RepeatedFailureWindow)
	assert.Equal(t, 3, thresholds.RepeatedFailureMinimum)
	assert.Equal(t, float64(4), thresholds.HighCostAcceptedMultiplier)
	assert.Equal(t, float64(2), thresholds.HighCostAcceptedMinimumUSD)
	assert.Equal(t, 50000, thresholds.VelocitySpikeTokens)
	assert.Equal(t, float64(60), thresholds.VelocitySpikeSeconds)
}

func TestDetect(t *testing.T) {
	thresholds := DefaultThresholds()
	now := time.Now()

	tests := []struct {
		name          string
		event         domain.TokenEvent
		prior         []domain.TokenEvent
		expectedTypes []domain.AnomalyType
	}{
		{
			name: "Happy Path - No Anomalies",
			event: domain.TokenEvent{
				EventID:         "evt_1",
				Timestamp:       now,
				WorkerID:        "worker_1",
				TotalTokens:     100,
				LatencyMs:       50,
				CostEstimateUSD: floatPtr(0.01),
				OutputStatus:    domain.OutputSucceeded,
			},
			prior: []domain.TokenEvent{
				{CostEstimateUSD: floatPtr(0.01), TotalTokens: 100, LatencyMs: 50, OutputStatus: domain.OutputSucceeded, WorkerID: "worker_1"},
				{CostEstimateUSD: floatPtr(0.01), TotalTokens: 100, LatencyMs: 50, OutputStatus: domain.OutputSucceeded, WorkerID: "worker_1"},
				{CostEstimateUSD: floatPtr(0.01), TotalTokens: 100, LatencyMs: 50, OutputStatus: domain.OutputSucceeded, WorkerID: "worker_1"},
			},
			expectedTypes: nil,
		},
		{
			name: "Unknown Model Pricing Anomaly",
			event: domain.TokenEvent{
				EventID:          "evt_1",
				Timestamp:        now,
				CostIsDegraded:   true,
				CostDegradedCode: "unknown_model_pricing",
			},
			prior:         nil,
			expectedTypes: []domain.AnomalyType{domain.AnomalyUnknownModelPricing},
		},
		{
			name: "Spend Spike Anomaly",
			event: domain.TokenEvent{
				EventID:         "evt_1",
				Timestamp:       now,
				CostEstimateUSD: floatPtr(5.0), // Mean is 1.0, mult is 3, min is 1. 5 > max(1, 3) -> Spike
			},
			prior: []domain.TokenEvent{
				{CostEstimateUSD: floatPtr(1.0)},
				{CostEstimateUSD: floatPtr(1.0)},
				{CostEstimateUSD: floatPtr(1.0)},
			},
			expectedTypes: []domain.AnomalyType{domain.AnomalySpendSpike},
		},
		{
			name: "Spend Spike Minimum Not Reached",
			event: domain.TokenEvent{
				EventID:         "evt_1",
				Timestamp:       now,
				CostEstimateUSD: floatPtr(0.5), // Mean is 0.1, mult is 3, min is 1. 0.5 < max(1, 0.3) -> No Spike
			},
			prior: []domain.TokenEvent{
				{CostEstimateUSD: floatPtr(0.1)},
				{CostEstimateUSD: floatPtr(0.1)},
				{CostEstimateUSD: floatPtr(0.1)},
			},
			expectedTypes: nil,
		},
		{
			name: "Token Spike Anomaly",
			event: domain.TokenEvent{
				EventID:     "evt_1",
				Timestamp:   now,
				TotalTokens: 40000, // Mean 1000, mult 3, min 10000. 40000 > max(10000, 3000) -> Spike
			},
			prior: []domain.TokenEvent{
				{TotalTokens: 1000},
				{TotalTokens: 1000},
				{TotalTokens: 1000},
			},
			expectedTypes: []domain.AnomalyType{domain.AnomalyTokenSpike},
		},
		{
			name: "Velocity Spike Anomaly",
			event: domain.TokenEvent{
				EventID:     "evt_1",
				Timestamp:   now,
				WorkerID:    "worker_1",
				TotalTokens: 20000,
			},
			prior: []domain.TokenEvent{
				{WorkerID: "worker_1", TotalTokens: 20000, Timestamp: now.Add(-10 * time.Second)},
				{WorkerID: "worker_1", TotalTokens: 20000, Timestamp: now.Add(-20 * time.Second)},  // Total 60000 > 50000
				{WorkerID: "worker_2", TotalTokens: 50000, Timestamp: now.Add(-10 * time.Second)},  // Different worker
				{WorkerID: "worker_1", TotalTokens: 20000, Timestamp: now.Add(-100 * time.Second)}, // Outside window
			},
			expectedTypes: []domain.AnomalyType{domain.AnomalyVelocitySpike},
		},
		{
			name: "Latency Spike Anomaly",
			event: domain.TokenEvent{
				EventID:   "evt_1",
				Timestamp: now,
				LatencyMs: 15000, // Mean 1000, mult 3, min 10000. 15000 > max(10000, 3000) -> Spike
			},
			prior: []domain.TokenEvent{
				{LatencyMs: 1000},
				{LatencyMs: 1000},
				{LatencyMs: 1000},
			},
			expectedTypes: []domain.AnomalyType{domain.AnomalyLatencySpike},
		},
		{
			name: "Repeated Failed Outputs Anomaly",
			event: domain.TokenEvent{
				EventID:      "evt_1",
				Timestamp:    now,
				WorkerID:     "worker_1",
				OutputStatus: domain.OutputFailed,
			},
			prior: []domain.TokenEvent{
				{WorkerID: "worker_1", OutputStatus: domain.OutputRejected},
				{WorkerID: "worker_1", OutputStatus: domain.OutputFailed},
				{WorkerID: "worker_1", OutputStatus: domain.OutputSucceeded},
				{WorkerID: "worker_2", OutputStatus: domain.OutputFailed}, // Different worker
			},
			expectedTypes: []domain.AnomalyType{domain.AnomalyRepeatedFailedOutputs},
		},
		{
			name: "High Cost Per Accepted Output Anomaly",
			event: domain.TokenEvent{
				EventID:         "evt_1",
				Timestamp:       now,
				CostEstimateUSD: floatPtr(5.0), // Mean 1.0, mult 4, min 2. 5 > max(2, 4) -> Anomaly
				OutputStatus:    domain.OutputAccepted,
				ReviewScore:     floatPtr(0.9),
			},
			prior: []domain.TokenEvent{
				{CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputAccepted, ReviewScore: floatPtr(0.9)},
				{CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputSucceeded, ReviewScore: floatPtr(0.9)},
				{CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputAccepted, ReviewScore: floatPtr(0.8)},
				{CostEstimateUSD: floatPtr(10.0), OutputStatus: domain.OutputFailed}, // Ignored
			},
			expectedTypes: []domain.AnomalyType{domain.AnomalyHighCostPerAcceptedOutput},
		},
		{
			name: "Multiple Anomalies",
			event: domain.TokenEvent{
				EventID:          "evt_1",
				Timestamp:        now,
				WorkerID:         "worker_1",
				CostIsDegraded:   true,
				CostDegradedCode: "unknown_model_pricing",
				TotalTokens:      60000, // Velocity spike
			},
			prior: nil, // Prior not needed for these two
			expectedTypes: []domain.AnomalyType{
				domain.AnomalyUnknownModelPricing,
				domain.AnomalyVelocitySpike,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signals := Detect(tt.event, tt.prior, now, thresholds)
			if len(tt.expectedTypes) == 0 {
				assert.Empty(t, signals)
			} else {
				assert.Len(t, signals, len(tt.expectedTypes))

				// Extract detected types
				var detectedTypes []domain.AnomalyType
				for _, sig := range signals {
					detectedTypes = append(detectedTypes, sig.Type)
				}

				// Assert that all expected types are present
				for _, expected := range tt.expectedTypes {
					assert.Contains(t, detectedTypes, expected)
				}
			}
		})
	}
}
