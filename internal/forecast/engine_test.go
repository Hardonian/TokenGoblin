package forecast

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
)

func ptr(f float64) *float64 { return &f }

func TestForecastMonthlySpend(t *testing.T) {
	eng := NewEngine()
	now := time.Now().UTC()

	var events []domain.TokenEvent
	// Simulate 15 days of spending: $10/day
	for d := 0; d < 15; d++ {
		for e := 0; e < 10; e++ {
			events = append(events, domain.TokenEvent{
				TenantID:        "t1",
				CostEstimateUSD: ptr(1.00),
				Timestamp:       now.AddDate(0, 0, -d),
				OutputStatus:    domain.OutputAccepted,
			})
		}
	}

	forecast := eng.ForecastMonthlySpend("t1", events)
	assert.Equal(t, "t1", forecast.TenantID)
	assert.True(t, forecast.CurrentMonthSpend > 0, "should have current spend")
	assert.True(t, len(forecast.Forecasts) > 0, "should have forecast snapshots")
	assert.True(t, forecast.Forecasts[0].PredictedCostUSD > 0, "should predict future spend")
	assert.True(t, forecast.Forecasts[0].ConfidenceLower <= forecast.Forecasts[0].PredictedCostUSD, "lower bound <= prediction")
	assert.True(t, forecast.Forecasts[0].ConfidenceUpper >= forecast.Forecasts[0].PredictedCostUSD, "upper bound >= prediction")
}

func TestEvaluateBudget(t *testing.T) {
	eng := NewEngine()

	tests := []struct {
		name       string
		limit      float64
		spend      float64
		threshold  float64
		wantStatus string
	}{
		{"healthy", 1000, 500, 0.8, domain.BudgetStatusHealthy},
		{"warning", 1000, 850, 0.8, domain.BudgetStatusWarning},
		{"critical", 1000, 920, 0.8, domain.BudgetStatusCritical},
		{"exceeded", 1000, 1100, 0.8, domain.BudgetStatusExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := domain.Budget{
				BudgetID:          "b1",
				TenantID:          "t1",
				LimitUSD:          tt.limit,
				AlertThresholdPct: tt.threshold,
			}
			result := eng.EvaluateBudget(budget, tt.spend)
			assert.Equal(t, tt.wantStatus, result.Status)
		})
	}
}

func TestGenerateScorecard(t *testing.T) {
	eng := NewEngine()
	now := time.Now().UTC()

	var events []domain.TokenEvent
	for i := 0; i < 100; i++ {
		status := domain.OutputAccepted
		if i < 20 {
			status = domain.OutputFailed
		}
		events = append(events, domain.TokenEvent{
			TenantID:        "t1",
			WorkerID:        "w1",
			CostEstimateUSD: ptr(0.10),
			OutputStatus:    status,
			Timestamp:       now.AddDate(0, 0, -1),
		})
	}

	scorecard := eng.GenerateScorecard("t1", events, 2.0)
	assert.Equal(t, "t1", scorecard.TenantID)
	assert.True(t, scorecard.AIMaturityScore > 0, "should have maturity score")
	assert.True(t, scorecard.AIMaturityScore <= 100, "maturity score should be <= 100")
	assert.InDelta(t, 0.80, scorecard.AvgAcceptanceRate, 0.01)
	assert.Equal(t, 100, scorecard.TotalEvents30d)
	assert.InDelta(t, 10.0, scorecard.TotalSpend30d, 0.01)
}

func TestCalculateStdDev(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		mean     float64
		expected float64
	}{
		{
			name:     "Empty values",
			values:   []float64{},
			mean:     10.0,
			expected: 10.0 * 0.3, // 30% uncertainty default
		},
		{
			name:     "Single value",
			values:   []float64{5.0},
			mean:     5.0,
			expected: 5.0 * 0.3, // 30% uncertainty default
		},
		{
			name:     "Zero variance",
			values:   []float64{10.0, 10.0, 10.0},
			mean:     10.0,
			expected: 0.0,
		},
		{
			name:     "Normal variance",
			values:   []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0},
			mean:     5.0,
			expected: 2.138, // Roughly 2.138
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStdDev(tt.values, tt.mean)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}
