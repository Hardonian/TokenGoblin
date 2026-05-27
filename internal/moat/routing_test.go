package moat

import (
	"reflect"
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func floatPtr(f float64) *float64 {
	return &f
}

func TestRecommendRoutes(t *testing.T) {
	tests := []struct {
		name   string
		events []domain.TokenEvent
		want   []domain.RoutingRecommendation
	}{
		{
			name:   "No events",
			events: []domain.TokenEvent{},
			want:   []domain.RoutingRecommendation{},
		},
		{
			name: "Single model per category",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "cat1",
					ModelID:         "model1",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1),
				},
				{
					TaskCategory:    "cat1",
					ModelID:         "model1",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1),
				},
			},
			want: []domain.RoutingRecommendation{},
		},
		{
			name: "Two models, significant savings",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "cat1",
					ModelID:         "best_model",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.01), // bestCostPer = 0.01
				},
				{
					TaskCategory:    "cat1",
					ModelID:         "worst_model",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1), // worstCostPer = 0.1
				},
				{
					TaskCategory:    "cat1",
					ModelID:         "worst_model",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1),
				},
				// worstTotalCost = 0.2
				// count(worst_model) = 2
				// projectedCost = 2 * 0.01 = 0.02
				// savings = 0.2 - 0.02 = 0.18 > 0.01
			},
			want: []domain.RoutingRecommendation{
				{
					TaskCategory:        "cat1",
					CurrentModel:        "worst_model",
					RecommendedModel:    "best_model",
					EstimatedSavingsUSD: 0.18,
					Reason:              "Routing this task to best_model instead of worst_model will save $0.18 with zero latency penalty.",
				},
			},
		},
		{
			name: "Two models, insignificant savings",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "cat1",
					ModelID:         "best_model",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1), // bestCostPer = 0.1
				},
				{
					TaskCategory:    "cat1",
					ModelID:         "worst_model",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.101), // worstCostPer = 0.101
				},
				{
					TaskCategory:    "cat1",
					ModelID:         "worst_model",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.101),
				},
				// worstTotalCost = 0.202
				// count(worst_model) = 2
				// projectedCost = 2 * 0.1 = 0.2
				// savings = 0.202 - 0.2 = 0.002 < 0.01 (insignificant)
			},
			want: []domain.RoutingRecommendation{},
		},
		{
			name: "Multiple categories",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "cat1",
					ModelID:         "best_model_1",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.01),
				},
				{
					TaskCategory:    "cat1",
					ModelID:         "worst_model_1",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1),
				},
				{
					TaskCategory:    "cat2",
					ModelID:         "best_model_2",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.05),
				},
				{
					TaskCategory:    "cat2",
					ModelID:         "worst_model_2",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.5),
				},
			},
			want: []domain.RoutingRecommendation{
				{
					TaskCategory:        "cat1",
					CurrentModel:        "worst_model_1",
					RecommendedModel:    "best_model_1",
					EstimatedSavingsUSD: 0.09,
					Reason:              "Routing this task to best_model_1 instead of worst_model_1 will save $0.09 with zero latency penalty.",
				},
				{
					TaskCategory:        "cat2",
					CurrentModel:        "worst_model_2",
					RecommendedModel:    "best_model_2",
					EstimatedSavingsUSD: 0.45,
					Reason:              "Routing this task to best_model_2 instead of worst_model_2 will save $0.45 with zero latency penalty.",
				},
			},
		},
		{
			name: "Ignored events",
			events: []domain.TokenEvent{
				// Missing TaskCategory
				{
					ModelID:         "model1",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1),
				},
				// Missing ModelID
				{
					TaskCategory:    "cat1",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.1),
				},
				// Missing CostEstimateUSD
				{
					TaskCategory: "cat1",
					ModelID:      "model1",
					OutputStatus: domain.OutputAccepted,
				},
				// OutputStatus neither accepted nor succeeded
				{
					TaskCategory:    "cat1",
					ModelID:         "model1",
					OutputStatus:    domain.OutputFailed,
					CostEstimateUSD: floatPtr(0.1),
				},
				// Valid events for comparison
				{
					TaskCategory:    "cat1",
					ModelID:         "best_model",
					OutputStatus:    domain.OutputAccepted,
					CostEstimateUSD: floatPtr(0.01),
				},
				{
					TaskCategory:    "cat1",
					ModelID:         "worst_model",
					OutputStatus:    domain.OutputSucceeded,
					CostEstimateUSD: floatPtr(0.1),
				},
			},
			want: []domain.RoutingRecommendation{
				{
					TaskCategory:        "cat1",
					CurrentModel:        "worst_model",
					RecommendedModel:    "best_model",
					EstimatedSavingsUSD: 0.09,
					Reason:              "Routing this task to best_model instead of worst_model will save $0.09 with zero latency penalty.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RecommendRoutes(tt.events)
			// For map iteration order, the order of recommendations might vary if there are multiple categories.
			// But for our tests, we either have 0, 1, or multiple. Let's just compare them ignoring order.
			if len(got) != len(tt.want) {
				t.Errorf("RecommendRoutes() = %v, want %v", got, tt.want)
			} else {
				for _, w := range tt.want {
					found := false
					for _, g := range got {
						// Account for floating point precision in EstimatedSavingsUSD
						// The reason string format contains %.2f, so it matches exactly
						if reflect.DeepEqual(g, w) || (g.TaskCategory == w.TaskCategory &&
							g.CurrentModel == w.CurrentModel &&
							g.RecommendedModel == w.RecommendedModel &&
							g.Reason == w.Reason &&
							abs(g.EstimatedSavingsUSD-w.EstimatedSavingsUSD) < 0.000001) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("RecommendRoutes() missing expected recommendation: %v\nGot: %v", w, got)
					}
				}
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
