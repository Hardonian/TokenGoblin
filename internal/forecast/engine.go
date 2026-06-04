package forecast

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

// Engine implements Layer 7 spend forecasting with linear extrapolation,
// confidence intervals, and budget tracking.
type Engine struct{}

// NewEngine creates a new forecasting engine instance.
func NewEngine() *Engine {
	return &Engine{}
}

// ═══════════════════════════════════════════════════════════
// Spend Forecasting
// ═══════════════════════════════════════════════════════════

// ForecastMonthlySpend predicts the current month's total spend based on
// observed daily spending patterns.
func (e *Engine) ForecastMonthlySpend(tenantID string, events []domain.TokenEvent) domain.SpendForecast {
	now := time.Now().UTC()
	currentYear, currentMonth, _ := now.Date()
	monthStart := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)
	daysInMonth := monthEnd.Sub(monthStart).Hours() / 24
	daysElapsed := now.Sub(monthStart).Hours() / 24
	if daysElapsed < 1 {
		daysElapsed = 1
	}
	daysRemaining := daysInMonth - daysElapsed

	// Build daily spend data
	dailyMap := make(map[string]*dailyAccum)
	for i := range events {
		ev := &events[i]
		if ev.Timestamp.Before(monthStart) || !ev.Timestamp.Before(monthEnd) {
			continue
		}
		dateStr := ev.Timestamp.Format("2006-01-02")
		da, ok := dailyMap[dateStr]
		if !ok {
			da = &dailyAccum{}
			dailyMap[dateStr] = da
		}
		if ev.CostEstimateUSD != nil {
			da.cost += *ev.CostEstimateUSD
		}
		da.events++
	}

	// Convert to sorted daily trend
	var dailyTrend []domain.DailySpend
	currentSpend := 0.0
	totalEvents := 0
	for dateStr, da := range dailyMap {
		dailyTrend = append(dailyTrend, domain.DailySpend{
			Date:    dateStr,
			CostUSD: da.cost,
			Events:  da.events,
		})
		currentSpend += da.cost
		totalEvents += da.events
	}
	sort.Slice(dailyTrend, func(i, j int) bool {
		return dailyTrend[i].Date < dailyTrend[j].Date
	})

	// Calculate daily average and standard deviation
	dailyAvg := 0.0
	if daysElapsed > 0 {
		dailyAvg = currentSpend / daysElapsed
	}

	dailyValues := make([]float64, 0, len(dailyMap))
	for _, da := range dailyMap {
		dailyValues = append(dailyValues, da.cost)
	}
	stdDev := calculateStdDev(dailyValues, dailyAvg)

	// Linear forecast
	predictedTotal := currentSpend + dailyAvg*daysRemaining

	// 80% confidence interval (±1.28 standard deviations per day × √remaining days)
	uncertainty := 1.28 * stdDev * math.Sqrt(daysRemaining)
	confidenceLower := math.Max(currentSpend, predictedTotal-uncertainty)
	confidenceUpper := predictedTotal + uncertainty

	// Generate forecast for remaining days
	var futureDaily []domain.DailySpend
	for d := 1; d <= int(daysRemaining)+1; d++ {
		futureDate := now.AddDate(0, 0, d)
		if !futureDate.Before(monthEnd) {
			break
		}
		futureDaily = append(futureDaily, domain.DailySpend{
			Date:     futureDate.Format("2006-01-02"),
			CostUSD:  dailyAvg,
			Events:   totalEvents / max(len(dailyMap), 1),
			IsFuture: true,
		})
	}
	dailyTrend = append(dailyTrend, futureDaily...)

	monthlyForecast := domain.ForecastSnapshot{
		ForecastID:       fmt.Sprintf("fc_%s_%s", tenantID, now.Format("20060102")),
		TenantID:         tenantID,
		GeneratedAt:      now,
		ForecastType:     domain.ForecastMonthly,
		PeriodStart:      monthStart,
		PeriodEnd:        monthEnd,
		PredictedCostUSD: roundTo(predictedTotal, 2),
		ConfidenceLower:  roundTo(confidenceLower, 2),
		ConfidenceUpper:  roundTo(confidenceUpper, 2),
		ModelUsed:        domain.ForecastModelLinear,
		InputEventCount:  totalEvents,
		InputDays:        int(daysElapsed),
	}

	// Previous month comparison
	prevMonthStart := monthStart.AddDate(0, -1, 0)
	var prevMonthSpend float64
	for i := range events {
		ev := &events[i]
		if !ev.Timestamp.Before(prevMonthStart) && ev.Timestamp.Before(monthStart) {
			if ev.CostEstimateUSD != nil {
				prevMonthSpend += *ev.CostEstimateUSD
			}
		}
	}

	var mom *domain.MonthComparison
	if prevMonthSpend > 0 {
		changePct := ((predictedTotal - prevMonthSpend) / prevMonthSpend) * 100
		trend := "stable"
		if changePct > 10 {
			trend = "increasing"
		} else if changePct < -10 {
			trend = "decreasing"
		}
		mom = &domain.MonthComparison{
			CurrentMonthUSD:  roundTo(predictedTotal, 2),
			PreviousMonthUSD: roundTo(prevMonthSpend, 2),
			ChangePercent:    roundTo(changePct, 1),
			Trend:            trend,
		}
	}

	return domain.SpendForecast{
		TenantID:          tenantID,
		GeneratedAt:       now,
		CurrentMonthSpend: roundTo(currentSpend, 2),
		DaysElapsed:       int(daysElapsed),
		DaysRemaining:     int(daysRemaining),
		Forecasts:         []domain.ForecastSnapshot{monthlyForecast},
		DailyTrend:        dailyTrend,
		MonthOverMonth:    mom,
	}
}

// ═══════════════════════════════════════════════════════════
// Budget Tracking
// ═══════════════════════════════════════════════════════════

// EvaluateBudget checks current spend against a budget and returns status.
func (e *Engine) EvaluateBudget(budget domain.Budget, currentSpend float64) domain.Budget {
	budget.CurrentSpendUSD = roundTo(currentSpend, 2)

	if budget.LimitUSD > 0 {
		budget.UtilizationPct = roundTo((currentSpend/budget.LimitUSD)*100, 1)
	}

	switch {
	case budget.UtilizationPct >= 100:
		budget.Status = domain.BudgetStatusExceeded
	case budget.UtilizationPct >= 90:
		budget.Status = domain.BudgetStatusCritical
	case budget.UtilizationPct >= budget.AlertThresholdPct*100:
		budget.Status = domain.BudgetStatusWarning
	default:
		budget.Status = domain.BudgetStatusHealthy
	}

	return budget
}

// ═══════════════════════════════════════════════════════════
// Executive Scorecard
// ═══════════════════════════════════════════════════════════

// GenerateScorecard produces an executive AI scorecard for leadership.
func (e *Engine) GenerateScorecard(tenantID string, events []domain.TokenEvent, wasteUSD float64) domain.ExecutiveScorecard {
	now := time.Now().UTC()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	var totalSpend float64
	var totalEvents int
	var acceptedCount int
	workerSet := make(map[string]struct{})

	for i := range events {
		ev := &events[i]
		if ev.Timestamp.Before(thirtyDaysAgo) {
			continue
		}
		totalEvents++
		if ev.CostEstimateUSD != nil {
			totalSpend += *ev.CostEstimateUSD
		}
		if ev.OutputStatus == domain.OutputAccepted || ev.OutputStatus == domain.OutputSucceeded {
			acceptedCount++
		}
		workerSet[ev.WorkerID] = struct{}{}
	}

	acceptanceRate := 0.0
	if totalEvents > 0 {
		acceptanceRate = float64(acceptedCount) / float64(totalEvents)
	}
	costPerOutcome := 0.0
	if acceptedCount > 0 {
		costPerOutcome = totalSpend / float64(acceptedCount)
	}

	// AI Maturity Score (0-100)
	// Based on: event volume, acceptance rate, waste ratio, worker diversity
	maturity := calculateMaturityScore(totalEvents, acceptanceRate, wasteUSD, totalSpend, len(workerSet))

	// Effectiveness Index: accepted outputs per dollar spent
	effectiveness := 0.0
	if totalSpend > 0 {
		effectiveness = float64(acceptedCount) / totalSpend
	}

	// ROI Index: value generated vs. cost (simplified as acceptance rate × efficiency)
	roiIndex := acceptanceRate * effectiveness * 100

	return domain.ExecutiveScorecard{
		TenantID:           tenantID,
		GeneratedAt:        now,
		AIMaturityScore:    maturity,
		AIEffectivenessIdx: roundTo(effectiveness, 4),
		AIAdoptionVelocity: float64(totalEvents) / 30.0,
		AIROIIndex:         roundTo(roiIndex, 2),
		TotalSpend30d:      roundTo(totalSpend, 2),
		TotalEvents30d:     totalEvents,
		ActiveWorkers:      len(workerSet),
		AvgAcceptanceRate:  roundTo(acceptanceRate, 4),
		CostPerOutcome:     roundTo(costPerOutcome, 4),
		WasteDetected30d:   roundTo(wasteUSD, 2),
	}
}

// ═══════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════

type dailyAccum struct {
	cost   float64
	events int
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return mean * 0.3 // Default 30% uncertainty with insufficient data
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func calculateMaturityScore(events int, acceptanceRate, wasteUSD, totalSpend float64, workers int) int {
	score := 0

	// Volume maturity (0-25 points)
	switch {
	case events > 10000:
		score += 25
	case events > 1000:
		score += 20
	case events > 100:
		score += 15
	case events > 10:
		score += 10
	case events > 0:
		score += 5
	}

	// Quality maturity (0-25 points)
	score += int(acceptanceRate * 25)

	// Efficiency maturity (0-25 points) — less waste = more mature
	wasteRatio := 0.0
	if totalSpend > 0 {
		wasteRatio = wasteUSD / totalSpend
	}
	efficiencyScore := int((1.0 - math.Min(wasteRatio, 1.0)) * 25)
	score += efficiencyScore

	// Adoption maturity (0-25 points) — more diverse usage = more mature
	switch {
	case workers > 20:
		score += 25
	case workers > 10:
		score += 20
	case workers > 5:
		score += 15
	case workers > 2:
		score += 10
	case workers > 0:
		score += 5
	}

	if score > 100 {
		score = 100
	}
	return score
}

func roundTo(value float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(value*pow) / pow
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
