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
		{"minimum possible score", 0, 0.0, 100.0, 100.0, 0, 0},
		{"maximum possible score", 10001, 1.0, 0.0, 100.0, 21, 100},
		{"volume tier 1 (>0)", 5, 0.0, 100.0, 100.0, 0, 5},
		{"volume tier 2 (>10)", 15, 0.0, 100.0, 100.0, 0, 10},
		{"volume tier 3 (>100)", 150, 0.0, 100.0, 100.0, 0, 15},
		{"volume tier 4 (>1000)", 1500, 0.0, 100.0, 100.0, 0, 20},
		{"quality tier partial", 0, 0.5, 100.0, 100.0, 0, 12}, // 0.5 * 25 = 12.5 -> 12
		{"efficiency perfect", 0, 0.0, 0.0, 100.0, 0, 25},
		{"efficiency partial", 0, 0.0, 50.0, 100.0, 0, 12}, // (1 - 50/100) * 25 = 12.5 -> 12
		{"adoption tier 1 (>0)", 0, 0.0, 100.0, 100.0, 1, 5},
		{"adoption tier 2 (>2)", 0, 0.0, 100.0, 100.0, 3, 10},
		{"adoption tier 3 (>5)", 0, 0.0, 100.0, 100.0, 6, 15},
		{"adoption tier 4 (>10)", 0, 0.0, 100.0, 100.0, 11, 20},
		{"score cap at 100", 15000, 1.5, 0.0, 100.0, 30, 100}, // > 100 theoretical score, but capped
		{"zero spend means no efficiency penalty", 0, 0.0, 0.0, 0.0, 0, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateMaturityScore(tt.events, tt.acceptanceRate, tt.wasteUSD, tt.totalSpend, tt.workers)
			assert.Equal(t, tt.wantScore, got)
		})
	}
}
