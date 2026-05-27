package moat_test

import (
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/moat"
)

func floatPtr(v float64) *float64 {
	return &v
}

func TestRecommendRoutes(t *testing.T) {
	tests := []struct {
		name     string
		events   []domain.TokenEvent
		expected []domain.RoutingRecommendation
	}{
		{
			name:     "empty events",
			events:   []domain.TokenEvent{},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "missing required fields",
			events: []domain.TokenEvent{
				{ModelID: "gpt-4", CostEstimateUSD: floatPtr(0.1), OutputStatus: domain.OutputAccepted},                                    // Missing TaskCategory
				{TaskCategory: "summarization", CostEstimateUSD: floatPtr(0.1), OutputStatus: domain.OutputAccepted},                     // Missing ModelID
				{TaskCategory: "summarization", ModelID: "gpt-4", OutputStatus: domain.OutputAccepted},                                   // Missing CostEstimateUSD
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "ignored output statuses",
			events: []domain.TokenEvent{
				{TaskCategory: "cat1", ModelID: "model1", CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputFailed},
				{TaskCategory: "cat1", ModelID: "model1", CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputRejected},
				{TaskCategory: "cat1", ModelID: "model1", CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputPending},
				{TaskCategory: "cat1", ModelID: "model2", CostEstimateUSD: floatPtr(0.1), OutputStatus: domain.OutputFailed},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "only one model in category",
			events: []domain.TokenEvent{
				{TaskCategory: "cat1", ModelID: "model1", CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "cat1", ModelID: "model1", CostEstimateUSD: floatPtr(2.0), OutputStatus: domain.OutputSucceeded},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "insignificant savings",
			events: []domain.TokenEvent{
				{TaskCategory: "cat1", ModelID: "expensive", CostEstimateUSD: floatPtr(0.105), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "cat1", ModelID: "cheap", CostEstimateUSD: floatPtr(0.100), OutputStatus: domain.OutputAccepted},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "valid recommendation",
			events: []domain.TokenEvent{
				{TaskCategory: "cat1", ModelID: "expensive", CostEstimateUSD: floatPtr(1.5), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "cat1", ModelID: "expensive", CostEstimateUSD: floatPtr(1.5), OutputStatus: domain.OutputSucceeded}, // Total cost 3.0, avg 1.5
				{TaskCategory: "cat1", ModelID: "cheap", CostEstimateUSD: floatPtr(0.5), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "cat1", ModelID: "cheap", CostEstimateUSD: floatPtr(0.5), OutputStatus: domain.OutputAccepted}, // Total cost 1.0, avg 0.5
			},
			// worstTotalCost = 3.0
			// projectedCost = count(expensive) * bestCostPer = 2 * 0.5 = 1.0
			// savings = 3.0 - 1.0 = 2.0
			expected: []domain.RoutingRecommendation{
				{
					TaskCategory:        "cat1",
					CurrentModel:        "expensive",
					RecommendedModel:    "cheap",
					EstimatedSavingsUSD: 2.0,
					Reason:              "Routing this task to cheap instead of expensive will save $2.00 with zero latency penalty.",
				},
			},
		},
		{
			name: "multiple categories",
			events: []domain.TokenEvent{
				// cat1 has recommendation
				{TaskCategory: "cat1", ModelID: "mod1", CostEstimateUSD: floatPtr(2.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "cat1", ModelID: "mod2", CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputAccepted},
				// cat2 has only one model
				{TaskCategory: "cat2", ModelID: "mod1", CostEstimateUSD: floatPtr(2.0), OutputStatus: domain.OutputAccepted},
				// cat3 has recommendation
				{TaskCategory: "cat3", ModelID: "mod3", CostEstimateUSD: floatPtr(10.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "cat3", ModelID: "mod4", CostEstimateUSD: floatPtr(1.0), OutputStatus: domain.OutputAccepted},
			},
			expected: []domain.RoutingRecommendation{
				{
					TaskCategory:        "cat1",
					CurrentModel:        "mod1",
					RecommendedModel:    "mod2",
					EstimatedSavingsUSD: 1.0,
					Reason:              "Routing this task to mod2 instead of mod1 will save $1.00 with zero latency penalty.",
				},
				{
					TaskCategory:        "cat3",
					CurrentModel:        "mod3",
					RecommendedModel:    "mod4",
					EstimatedSavingsUSD: 9.0,
					Reason:              "Routing this task to mod4 instead of mod3 will save $9.00 with zero latency penalty.",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := moat.RecommendRoutes(tc.events)

			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d recommendations, got %d", len(tc.expected), len(result))
			}

			// Since the order might not be deterministic for multiple categories
			// we verify each expected item exists
			for _, exp := range tc.expected {
				found := false
				for _, res := range result {
					if res.TaskCategory == exp.TaskCategory &&
						res.CurrentModel == exp.CurrentModel &&
						res.RecommendedModel == exp.RecommendedModel &&
						res.EstimatedSavingsUSD == exp.EstimatedSavingsUSD &&
						res.Reason == exp.Reason {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected recommendation not found: %+v", exp)
				}
			}
		})
	}
}
