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

func TestCalculateMaturityScore(t *testing.T) {
	tests := []struct {
		name           string
		events         int
		acceptanceRate float64
		wasteUSD       float64
		totalSpend     float64
		workers        int
		wantScore      int
	}{
		// Happy path / Maximum score (100)
		// events > 10000 (25), acceptanceRate 1.0 (25), wasteRatio 0.0 (25), workers > 20 (25) -> 100
		{"max score", 10001, 1.0, 0.0, 100.0, 21, 100},

		// Edge cases for 0 score
		{"zero score", 0, 0.0, 100.0, 100.0, 0, 0}, // wasteRatio 1.0 -> 0 efficiency points

		// Score capping at 100
		{"capped at 100", 20000, 1.5, 0.0, 100.0, 50, 100},

		// Volume maturity thresholds
		{"volume > 1000", 1001, 0.0, 100.0, 100.0, 0, 20},
		{"volume > 100", 101, 0.0, 100.0, 100.0, 0, 15},
		{"volume > 10", 11, 0.0, 100.0, 100.0, 0, 10},
		{"volume > 0", 1, 0.0, 100.0, 100.0, 0, 5},

		// Quality maturity
		{"quality mid", 0, 0.5, 100.0, 100.0, 0, 12}, // int(0.5 * 25) = 12

		// Efficiency maturity
		{"efficiency no spend", 0, 0.0, 10.0, 0.0, 0, 25},      // totalSpend 0 -> wasteRatio 0 -> 25 points
		{"efficiency high waste", 0, 0.0, 200.0, 100.0, 0, 0}, // wasteRatio 2.0 capped at 1.0 -> 0 points
		{"efficiency mid waste", 0, 0.0, 25.0, 100.0, 0, 18},  // wasteRatio 0.25 -> 75% of 25 = 18.75 -> 18

		// Adoption maturity thresholds
		{"adoption > 10", 0, 0.0, 100.0, 100.0, 11, 20},
		{"adoption > 5", 0, 0.0, 100.0, 100.0, 6, 15},
		{"adoption > 2", 0, 0.0, 100.0, 100.0, 3, 10},
		{"adoption > 0", 0, 0.0, 100.0, 100.0, 1, 5},

		// Mixed scenario
		// events 15 (10), acc 0.8 (20), waste 10/50 (20), workers 4 (10) -> 60
		{"mixed scenario", 15, 0.8, 10.0, 50.0, 4, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateMaturityScore(tt.events, tt.acceptanceRate, tt.wasteUSD, tt.totalSpend, tt.workers)
			assert.Equal(t, tt.wantScore, score)
		})
	}
}
