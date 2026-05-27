package productivity

import (
	"sort"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func BuildSummary(tenantID string, events []domain.TokenEvent, anomalies []domain.AnomalySignal, generatedAt time.Time) domain.ProductivitySummary {
	summary := domain.ProductivitySummary{
		SummaryID:      "latest",
		TenantID:       tenantID,
		GeneratedAt:    generatedAt.UTC(),
		CostByWorker:   []domain.WorkerBreakdown{},
		CostByCategory: []domain.CategoryBreakdown{},
		TopCostDrivers: []domain.CostDriver{},
	}

	if len(events) == 0 {
		summary.Degraded = append(summary.Degraded, domain.Issue{
			Code:    "no_data",
			Message: "No usage events exist for this tenant.",
		})
		return summary
	}

	workerStats := map[string]*workerAccumulator{}
	categoryStats := map[string]*categoryAccumulator{}
	modelStats := map[string]*driverAccumulator{}
	anomalyByWorker := map[string]int{}
	for _, signal := range anomalies {
		if signal.WorkerID != "" {
			anomalyByWorker[signal.WorkerID]++
		}
	}

	var latencyTotal float64
	var latencyCount int
	var acceptedReviewedCost float64
	var acceptedReviewedCount int

	for _, event := range events {
		summary.TotalEvents++
		if event.CostEstimateUSD == nil {
			summary.UnknownCostEventCount++
		} else {
			summary.KnownCostEventCount++
			summary.TotalCostUSD += *event.CostEstimateUSD
		}
		if isAccepted(event.OutputStatus) {
			summary.OutputCount++
			if event.ReviewScore != nil && event.CostEstimateUSD != nil {
				acceptedReviewedCost += *event.CostEstimateUSD
				acceptedReviewedCount++
			}
		}
		if event.LatencyMs > 0 {
			latencyTotal += float64(event.LatencyMs)
			latencyCount++
		}

		worker := workerStats[event.WorkerID]
		if worker == nil {
			worker = &workerAccumulator{workerID: event.WorkerID, workerName: event.WorkerName, now: generatedAt}
			workerStats[event.WorkerID] = worker
		}
		worker.add(event)

		category := event.TaskCategory
		if category == "" {
			category = "uncategorized"
		}
		cat := categoryStats[category]
		if cat == nil {
			cat = &categoryAccumulator{taskCategory: category}
			categoryStats[category] = cat
		}
		cat.add(event)

		modelKey := event.Provider + ":" + event.ModelID
		model := modelStats[modelKey]
		if model == nil {
			model = &driverAccumulator{key: modelKey, label: modelKey}
			modelStats[modelKey] = model
		}
		model.add(event)
	}

	if latencyCount > 0 {
		avg := latencyTotal / float64(latencyCount)
		summary.AvgLatencyMs = &avg
	}
	summary.AnomalyCount = len(anomalies)
	if acceptedReviewedCount > 0 {
		value := acceptedReviewedCost / float64(acceptedReviewedCount)
		summary.CostPerAcceptedOutputWithReview = &value
	}

	for _, item := range workerStats {
		breakdown := item.breakdown()
		breakdown.AnomalyCount = anomalyByWorker[item.workerID]
		summary.CostByWorker = append(summary.CostByWorker, breakdown)
		summary.TopCostDrivers = append(summary.TopCostDrivers, domain.CostDriver{
			Type:         "worker",
			Key:          breakdown.WorkerID,
			Label:        breakdown.WorkerName,
			TotalCostUSD: breakdown.TotalCostUSD,
			EventCount:   breakdown.EventCount,
		})
	}
	for _, item := range categoryStats {
		breakdown := item.breakdown()
		summary.CostByCategory = append(summary.CostByCategory, breakdown)
		summary.TopCostDrivers = append(summary.TopCostDrivers, domain.CostDriver{
			Type:         "category",
			Key:          breakdown.TaskCategory,
			Label:        breakdown.TaskCategory,
			TotalCostUSD: breakdown.TotalCostUSD,
			EventCount:   breakdown.EventCount,
		})
	}
	for _, item := range modelStats {
		summary.TopCostDrivers = append(summary.TopCostDrivers, domain.CostDriver{
			Type:         "model",
			Key:          item.key,
			Label:        item.label,
			TotalCostUSD: item.totalCost,
			EventCount:   item.eventCount,
		})
	}

	sort.Slice(summary.CostByWorker, func(i, j int) bool {
		return summary.CostByWorker[i].TotalCostUSD > summary.CostByWorker[j].TotalCostUSD
	})
	sort.Slice(summary.CostByCategory, func(i, j int) bool {
		return summary.CostByCategory[i].TotalCostUSD > summary.CostByCategory[j].TotalCostUSD
	})
	sort.Slice(summary.TopCostDrivers, func(i, j int) bool {
		if summary.TopCostDrivers[i].TotalCostUSD == summary.TopCostDrivers[j].TotalCostUSD {
			return summary.TopCostDrivers[i].Key < summary.TopCostDrivers[j].Key
		}
		return summary.TopCostDrivers[i].TotalCostUSD > summary.TopCostDrivers[j].TotalCostUSD
	})
	if len(summary.TopCostDrivers) > 5 {
		summary.TopCostDrivers = summary.TopCostDrivers[:5]
	}

	if summary.UnknownCostEventCount > 0 {
		summary.Degraded = append(summary.Degraded, domain.Issue{
			Code:    "unknown_pricing_present",
			Message: "Some usage events have unknown model pricing and are excluded from cost totals.",
		})
	}

	return summary
}

type workerAccumulator struct {
	workerID             string
	workerName           string
	eventCount           int
	outputCount          int
	failedCount          int
	totalTokens          int
	totalCost            float64
	unknownCostCount     int
	latencyTotal         float64
	latencyCount         int
	acceptedReviewedCost float64
	acceptedReviewed     int

	// For memory trend
	currentPeriodCost  float64
	currentPeriodCount int
	priorPeriodCost    float64
	priorPeriodCount   int
	now                time.Time
}

func (a *workerAccumulator) add(event domain.TokenEvent) {
	a.eventCount++
	a.totalTokens += event.TotalTokens
	if event.WorkerName != "" {
		a.workerName = event.WorkerName
	}
	if event.CostEstimateUSD == nil {
		a.unknownCostCount++
	} else {
		a.totalCost += *event.CostEstimateUSD
	}
	if isAccepted(event.OutputStatus) {
		a.outputCount++
		if event.ReviewScore != nil && event.CostEstimateUSD != nil {
			a.acceptedReviewedCost += *event.CostEstimateUSD
			a.acceptedReviewed++
		}
	}
	if isFailure(event.OutputStatus) {
		a.failedCount++
	}
	if event.LatencyMs > 0 {
		a.latencyTotal += float64(event.LatencyMs)
		a.latencyCount++
	}

	// Memory Trend (7-day vs prior 7-day)
	if isAccepted(event.OutputStatus) && event.CostEstimateUSD != nil {
		sevenDaysAgo := a.now.Add(-7 * 24 * time.Hour)
		fourteenDaysAgo := a.now.Add(-14 * 24 * time.Hour)
		
		if event.Timestamp.After(sevenDaysAgo) && !event.Timestamp.After(a.now) {
			a.currentPeriodCost += *event.CostEstimateUSD
			a.currentPeriodCount++
		} else if event.Timestamp.After(fourteenDaysAgo) && !event.Timestamp.After(sevenDaysAgo) {
			a.priorPeriodCost += *event.CostEstimateUSD
			a.priorPeriodCount++
		}
	}
}

func (a *workerAccumulator) breakdown() domain.WorkerBreakdown {
	breakdown := domain.WorkerBreakdown{
		WorkerID:              a.workerID,
		WorkerName:            a.workerName,
		EventCount:            a.eventCount,
		OutputCount:           a.outputCount,
		FailedOutputCount:     a.failedCount,
		TotalTokens:           a.totalTokens,
		TotalCostUSD:          a.totalCost,
		UnknownCostEventCount: a.unknownCostCount,
	}
	if breakdown.WorkerName == "" {
		breakdown.WorkerName = breakdown.WorkerID
	}
	if a.latencyCount > 0 {
		avg := a.latencyTotal / float64(a.latencyCount)
		breakdown.AvgLatencyMs = &avg
	}
	if a.acceptedReviewed > 0 {
		value := a.acceptedReviewedCost / float64(a.acceptedReviewed)
		breakdown.CostPerAcceptedOutputWithReview = &value
	}

	// Compute trend
	breakdown.Trend = "stable"
	var currentCostPer, priorCostPer float64
	if a.currentPeriodCount > 0 {
		currentCostPer = a.currentPeriodCost / float64(a.currentPeriodCount)
	}
	if a.priorPeriodCount > 0 {
		priorCostPer = a.priorPeriodCost / float64(a.priorPeriodCount)
	}

	if a.currentPeriodCount > 0 && a.priorPeriodCount > 0 {
		if currentCostPer > priorCostPer*1.1 {
			breakdown.Trend = "decaying"
		} else if currentCostPer < priorCostPer*0.9 {
			breakdown.Trend = "improving"
		}
	}

	breakdown.EfficiencyRating = "optimal"
	if breakdown.Trend == "decaying" {
		breakdown.EfficiencyRating = "degraded"
	}

	return breakdown
}

type categoryAccumulator struct {
	taskCategory string
	eventCount   int
	outputCount  int
	totalCost    float64
	latencyTotal float64
	latencyCount int
}

func (a *categoryAccumulator) add(event domain.TokenEvent) {
	a.eventCount++
	if isAccepted(event.OutputStatus) {
		a.outputCount++
	}
	if event.CostEstimateUSD != nil {
		a.totalCost += *event.CostEstimateUSD
	}
	if event.LatencyMs > 0 {
		a.latencyTotal += float64(event.LatencyMs)
		a.latencyCount++
	}
}

func (a *categoryAccumulator) breakdown() domain.CategoryBreakdown {
	breakdown := domain.CategoryBreakdown{
		TaskCategory: a.taskCategory,
		EventCount:   a.eventCount,
		OutputCount:  a.outputCount,
		TotalCostUSD: a.totalCost,
	}
	if a.latencyCount > 0 {
		avg := a.latencyTotal / float64(a.latencyCount)
		breakdown.AvgLatencyMs = &avg
	}
	return breakdown
}

type driverAccumulator struct {
	key        string
	label      string
	eventCount int
	totalCost  float64
}

func (a *driverAccumulator) add(event domain.TokenEvent) {
	a.eventCount++
	if event.CostEstimateUSD != nil {
		a.totalCost += *event.CostEstimateUSD
	}
}

func isAccepted(status domain.OutputStatus) bool {
	return status == domain.OutputAccepted || status == domain.OutputSucceeded
}

func isFailure(status domain.OutputStatus) bool {
	return status == domain.OutputFailed || status == domain.OutputRejected
}
