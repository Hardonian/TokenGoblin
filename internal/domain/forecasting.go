package domain

import "time"

// ═══════════════════════════════════════════════════════════
// Layer 7 — Forecasting Engine Types
// ═══════════════════════════════════════════════════════════

// ForecastType defines the forecasting time horizon.
type ForecastType string

const (
	ForecastDaily     ForecastType = "daily_spend"
	ForecastWeekly    ForecastType = "weekly_spend"
	ForecastMonthly   ForecastType = "monthly_spend"
	ForecastQuarterly ForecastType = "quarterly_spend"
	ForecastAnnual    ForecastType = "annual_spend"
)

// ForecastModel defines the algorithm used for prediction.
type ForecastModel string

const (
	ForecastModelLinear      ForecastModel = "linear"
	ForecastModelExponential ForecastModel = "exponential"
	ForecastModelSeasonal    ForecastModel = "seasonal"
	ForecastModelMovingAvg   ForecastModel = "moving_average"
)

// ForecastSnapshot stores a single point-in-time forecast.
type ForecastSnapshot struct {
	ForecastID       string        `json:"forecast_id"`
	TenantID         string        `json:"tenant_id"`
	GeneratedAt      time.Time     `json:"generated_at"`
	ForecastType     ForecastType  `json:"forecast_type"`
	PeriodStart      time.Time     `json:"period_start"`
	PeriodEnd        time.Time     `json:"period_end"`
	PredictedCostUSD float64       `json:"predicted_cost_usd"`
	ConfidenceLower  float64       `json:"confidence_lower"`
	ConfidenceUpper  float64       `json:"confidence_upper"`
	ModelUsed        ForecastModel `json:"model_used"`
	InputEventCount  int           `json:"input_event_count"`
	InputDays        int           `json:"input_days"`
}

// SpendForecast wraps multiple forecast snapshots for API response.
type SpendForecast struct {
	TenantID          string             `json:"tenant_id"`
	GeneratedAt       time.Time          `json:"generated_at"`
	CurrentMonthSpend float64            `json:"current_month_spend"`
	DaysElapsed       int                `json:"days_elapsed"`
	DaysRemaining     int                `json:"days_remaining"`
	Forecasts         []ForecastSnapshot `json:"forecasts"`
	DailyTrend        []DailySpend       `json:"daily_trend"`
	MonthOverMonth    *MonthComparison   `json:"month_over_month,omitempty"`
}

// DailySpend is a single data point in the daily spend trend.
type DailySpend struct {
	Date     string  `json:"date"`
	CostUSD  float64 `json:"cost_usd"`
	Events   int     `json:"events"`
	IsFuture bool    `json:"is_future"`
}

// MonthComparison compares current month vs previous month.
type MonthComparison struct {
	CurrentMonthUSD  float64 `json:"current_month_usd"`
	PreviousMonthUSD float64 `json:"previous_month_usd"`
	ChangePercent    float64 `json:"change_percent"`
	Trend            string  `json:"trend"`
}

// BudgetScopeType defines what scope a budget covers.
type BudgetScopeType string

const (
	BudgetScopeTenant  BudgetScopeType = "tenant"
	BudgetScopeTeam    BudgetScopeType = "team"
	BudgetScopeWorker  BudgetScopeType = "worker"
	BudgetScopeProject BudgetScopeType = "project"
)

// Budget defines a spending limit for a scope and time period.
type Budget struct {
	BudgetID          string          `json:"budget_id"`
	TenantID          string          `json:"tenant_id"`
	Name              string          `json:"name"`
	Type              string          `json:"type"`
	ScopeType         BudgetScopeType `json:"scope_type"`
	ScopeID           string          `json:"scope_id,omitempty"`
	LimitUSD          float64         `json:"limit_usd"`
	AlertThresholdPct float64         `json:"alert_threshold_pct"`
	CurrentSpendUSD   float64         `json:"current_spend_usd"`
	PeriodStart       time.Time       `json:"period_start"`
	PeriodEnd         time.Time       `json:"period_end"`
	IsActive          bool            `json:"is_active"`
	CreatedAt         time.Time       `json:"created_at"`
	UtilizationPct    float64         `json:"utilization_pct"`
	Status            string          `json:"status"`
}

// BudgetStatus constants.
const (
	BudgetStatusHealthy  = "healthy"
	BudgetStatusWarning  = "warning"
	BudgetStatusCritical = "critical"
	BudgetStatusExceeded = "exceeded"
)

// ROITrend represents return on AI investment over time.
type ROITrend struct {
	TenantID    string         `json:"tenant_id"`
	GeneratedAt time.Time      `json:"generated_at"`
	DataPoints  []ROIDataPoint `json:"data_points"`
}

// ROIDataPoint represents a single period's ROI calculation.
type ROIDataPoint struct {
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	TotalCostUSD    float64   `json:"total_cost_usd"`
	AcceptedOutputs int       `json:"accepted_outputs"`
	CostPerOutcome  float64   `json:"cost_per_outcome"`
	EfficiencyIndex float64   `json:"efficiency_index"`
}
