package moat

import (
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func floatPtr(f float64) *float64 {
	return &f
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
			name: "missing required fields ignored",
			events: []domain.TokenEvent{
				{
					ModelID:         "model-1",
					CostEstimateUSD: floatPtr(1.0),
					OutputStatus:    domain.OutputAccepted,
					// missing TaskCategory
				},
				{
					TaskCategory:    "cat-1",
					CostEstimateUSD: floatPtr(1.0),
					OutputStatus:    domain.OutputAccepted,
					// missing ModelID
				},
				{
					TaskCategory: "cat-1",
					ModelID:      "model-1",
					OutputStatus: domain.OutputAccepted,
					// missing CostEstimateUSD
				},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "failed status ignored",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "cat-1",
					ModelID:         "model-1",
					CostEstimateUSD: floatPtr(1.0),
					OutputStatus:    domain.OutputFailed,
				},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "single model per category yields no recommendation",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "cat-1",
					ModelID:         "model-1",
					CostEstimateUSD: floatPtr(1.0),
					OutputStatus:    domain.OutputAccepted,
				},
				{
					TaskCategory:    "cat-1",
					ModelID:         "model-1",
					CostEstimateUSD: floatPtr(2.0),
					OutputStatus:    domain.OutputAccepted,
				},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "savings under 0.01 yields no recommendation",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "cat-1",
					ModelID:         "model-cheap",
					CostEstimateUSD: floatPtr(0.100),
					OutputStatus:    domain.OutputAccepted,
				},
				{
					TaskCategory:    "cat-1",
					ModelID:         "model-expensive",
					CostEstimateUSD: floatPtr(0.105),
					OutputStatus:    domain.OutputAccepted,
				},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "savings over 0.01 yields recommendation",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "translation",
					ModelID:         "model-cheap",
					CostEstimateUSD: floatPtr(1.0),
					OutputStatus:    domain.OutputAccepted,
				},
				{
					TaskCategory:    "translation",
					ModelID:         "model-expensive",
					CostEstimateUSD: floatPtr(5.0),
					OutputStatus:    domain.OutputSucceeded,
				},
			},
			expected: []domain.RoutingRecommendation{
				{
					TaskCategory:        "translation",
					CurrentModel:        "model-expensive",
					RecommendedModel:    "model-cheap",
					EstimatedSavingsUSD: 4.0, // 5.0 - (1 * 1.0)
					Reason:              "Routing this task to model-cheap instead of model-expensive will save $4.00 with zero latency penalty.",
				},
			},
		},
		{
			name: "savings calculated correctly with multiple runs",
			events: []domain.TokenEvent{
				{
					TaskCategory:    "coding",
					ModelID:         "model-A",
					CostEstimateUSD: floatPtr(2.0),
					OutputStatus:    domain.OutputAccepted,
				},
				{
					TaskCategory:    "coding",
					ModelID:         "model-A",
					CostEstimateUSD: floatPtr(2.0),
					OutputStatus:    domain.OutputAccepted,
				},
				{
					TaskCategory:    "coding",
					ModelID:         "model-B",
					CostEstimateUSD: floatPtr(10.0),
					OutputStatus:    domain.OutputAccepted,
				},
				{
					TaskCategory:    "coding",
					ModelID:         "model-B",
					CostEstimateUSD: floatPtr(10.0),
					OutputStatus:    domain.OutputAccepted,
				},
			},
			expected: []domain.RoutingRecommendation{
				{
					TaskCategory:        "coding",
					CurrentModel:        "model-B",
					RecommendedModel:    "model-A",
					EstimatedSavingsUSD: 16.0, // worst total = 20.0, worst count = 2, best per = 2.0. savings = 20.0 - 4.0 = 16.0
					Reason:              "Routing this task to model-A instead of model-B will save $16.00 with zero latency penalty.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := RecommendRoutes(tt.events)

			if len(recs) != len(tt.expected) {
				t.Fatalf("expected %d recommendations, got %d", len(tt.expected), len(recs))
			}

			for i := range recs {
				if recs[i].TaskCategory != tt.expected[i].TaskCategory {
					t.Errorf("expected category %q, got %q", tt.expected[i].TaskCategory, recs[i].TaskCategory)
				}
				if recs[i].CurrentModel != tt.expected[i].CurrentModel {
					t.Errorf("expected current model %q, got %q", tt.expected[i].CurrentModel, recs[i].CurrentModel)
				}
				if recs[i].RecommendedModel != tt.expected[i].RecommendedModel {
					t.Errorf("expected recommended model %q, got %q", tt.expected[i].RecommendedModel, recs[i].RecommendedModel)
				}
				if recs[i].EstimatedSavingsUSD != tt.expected[i].EstimatedSavingsUSD {
					t.Errorf("expected savings %f, got %f", tt.expected[i].EstimatedSavingsUSD, recs[i].EstimatedSavingsUSD)
				}
				if recs[i].Reason != tt.expected[i].Reason {
					t.Errorf("expected reason %q, got %q", tt.expected[i].Reason, recs[i].Reason)
				}
			}
		})
	}
}
