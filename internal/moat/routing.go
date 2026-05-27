package moat

import (
	"fmt"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func RecommendRoutes(events []domain.TokenEvent) []domain.RoutingRecommendation {
	var recs []domain.RoutingRecommendation

	type stats struct {
		AcceptedCount int
		TotalCost     float64
	}

	modelStatsByCategory := make(map[string]map[string]*stats)

	for i := range events {
		event := &events[i] // Use pointer to avoid copying struct
		if event.TaskCategory == "" || event.ModelID == "" || event.CostEstimateUSD == nil {
			continue
		}
		if event.OutputStatus != domain.OutputAccepted && event.OutputStatus != domain.OutputSucceeded {
			continue
		}

		catMap := modelStatsByCategory[event.TaskCategory]
		if catMap == nil {
			catMap = make(map[string]*stats)
			modelStatsByCategory[event.TaskCategory] = catMap
		}

		stat := catMap[event.ModelID]
		if stat == nil {
			stat = &stats{}
			catMap[event.ModelID] = stat
		}

		stat.AcceptedCount++
		stat.TotalCost += *event.CostEstimateUSD
	}

	for category, models := range modelStatsByCategory {
		if len(models) < 2 {
			continue
		}

		var bestModel, worstModel string
		var bestCostPer, worstCostPer float64
		var worstTotalCost float64

		first := true
		for model, stat := range models {
			if stat.AcceptedCount == 0 {
				continue
			}
			costPer := stat.TotalCost / float64(stat.AcceptedCount)
			if first {
				bestModel, worstModel = model, model
				bestCostPer, worstCostPer = costPer, costPer
				worstTotalCost = stat.TotalCost
				first = false
				continue
			}

			if costPer < bestCostPer {
				bestCostPer = costPer
				bestModel = model
			}
			if costPer > worstCostPer {
				worstCostPer = costPer
				worstModel = model
				worstTotalCost = stat.TotalCost
			}
		}

		if bestModel != "" && worstModel != "" && bestModel != worstModel {
			count := models[worstModel].AcceptedCount
			projectedCost := float64(count) * bestCostPer
			savings := worstTotalCost - projectedCost

			if savings > 0.01 { // Only recommend if savings are notable
				recs = append(recs, domain.RoutingRecommendation{
					TaskCategory:        category,
					CurrentModel:        worstModel,
					RecommendedModel:    bestModel,
					EstimatedSavingsUSD: savings,
					EvidenceCount:       count,
					Basis:               "accepted_output_cost_per_task",
					Confidence:          confidenceFor(count),
					Status:              "open",
					Reason:              fmt.Sprintf("Recent accepted outputs for this task cost less on %s than %s; estimated savings are $%.2f for the observed workload.", bestModel, worstModel, savings),
				})
			}
		}
	}

	if recs == nil {
		recs = []domain.RoutingRecommendation{}
	}
	return recs
}

func confidenceFor(count int) string {
	if count >= 10 {
		return "high"
	}
	if count >= 3 {
		return "medium"
	}
	return "low"
}
