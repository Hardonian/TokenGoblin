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
		expected       int
	}{
		{"zeros", 0, 0.0, 0.0, 0.0, 0, 25}, // 0 (events) + 0 (quality) + 25 (efficiency, wasteRatio=0) + 0 (workers) = 25
		{"all maximums", 10001, 1.0, 0.0, 1000.0, 21, 100}, // 25 (events) + 25 (quality) + 25 (efficiency) + 25 (workers) = 100

		// Volume (assuming other factors are 0 except efficiency with wasteRatio=0 totalSpend=0 which yields 25)
		{"volume 1-10", 5, 0.0, 0.0, 0.0, 0, 30}, // 5 (volume) + 0 (quality) + 25 (efficiency) + 0 (workers) = 30
		{"volume 11-100", 15, 0.0, 0.0, 0.0, 0, 35}, // 10 (volume) + 0 (quality) + 25 (efficiency) + 0 (workers) = 35
		{"volume 101-1000", 105, 0.0, 0.0, 0.0, 0, 40}, // 15 (volume) + 0 (quality) + 25 (efficiency) + 0 (workers) = 40
		{"volume 1001-10000", 1005, 0.0, 0.0, 0.0, 0, 45}, // 20 (volume) + 0 (quality) + 25 (efficiency) + 0 (workers) = 45
		{"volume >10000", 10005, 0.0, 0.0, 0.0, 0, 50}, // 25 (volume) + 0 (quality) + 25 (efficiency) + 0 (workers) = 50

		// Quality
		{"quality 50%", 0, 0.5, 0.0, 0.0, 0, 37}, // 0 (volume) + 12 (quality) + 25 (efficiency) + 0 (workers) = 37

		// Efficiency
		{"efficiency 50% waste", 0, 0.0, 500.0, 1000.0, 0, 12}, // 0 (volume) + 0 (quality) + 12 (efficiency: 1 - 0.5 * 25) + 0 (workers) = 12
		{"efficiency 100% waste", 0, 0.0, 1000.0, 1000.0, 0, 0}, // 0 (volume) + 0 (quality) + 0 (efficiency: 1 - 1.0 * 25) + 0 (workers) = 0
		{"efficiency >100% waste", 0, 0.0, 1500.0, 1000.0, 0, 0}, // Math.min limits waste to 1.0 => 0

		// Adoption
		{"workers 1-2", 0, 0.0, 0.0, 0.0, 1, 30}, // 0 (volume) + 0 (quality) + 25 (efficiency) + 5 (workers) = 30
		{"workers 3-5", 0, 0.0, 0.0, 0.0, 3, 35}, // 0 (volume) + 0 (quality) + 25 (efficiency) + 10 (workers) = 35
		{"workers 6-10", 0, 0.0, 0.0, 0.0, 6, 40}, // 0 (volume) + 0 (quality) + 25 (efficiency) + 15 (workers) = 40
		{"workers 11-20", 0, 0.0, 0.0, 0.0, 11, 45}, // 0 (volume) + 0 (quality) + 25 (efficiency) + 20 (workers) = 45
		{"workers >20", 0, 0.0, 0.0, 0.0, 21, 50}, // 0 (volume) + 0 (quality) + 25 (efficiency) + 25 (workers) = 50

		{"capped at 100", 100000, 2.0, 0.0, 1000.0, 50, 100}, // > 100 should cap at 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateMaturityScore(tt.events, tt.acceptanceRate, tt.wasteUSD, tt.totalSpend, tt.workers)
			assert.Equal(t, tt.expected, score)
		})
	}
}
