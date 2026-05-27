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
					Reason:              fmt.Sprintf("Routing this task to %s instead of %s will save $%.2f with zero latency penalty.", bestModel, worstModel, savings),
				})
			}
		}
	}

	if recs == nil {
		recs = []domain.RoutingRecommendation{}
	}
	return recs
}
