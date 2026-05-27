package moat

import (
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func ptr(f float64) *float64 {
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
			name: "ignored due to missing fields",
			events: []domain.TokenEvent{
				{TaskCategory: "", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: nil, OutputStatus: domain.OutputAccepted},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "ignored due to status",
			events: []domain.TokenEvent{
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputFailed},
				{TaskCategory: "c1", ModelID: "m2", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputRejected},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "single model in category",
			events: []domain.TokenEvent{
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "happy path savings",
			events: []domain.TokenEvent{
				// Model 1: 2 accepted, total cost 2.0 -> 1.0 per
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
				// Model 2: 2 accepted, total cost 4.0 -> 2.0 per
				{TaskCategory: "c1", ModelID: "m2", CostEstimateUSD: ptr(2.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "m2", CostEstimateUSD: ptr(2.0), OutputStatus: domain.OutputAccepted},
			},
			// Worst: m2 (2.0 per, 4.0 total), Best: m1 (1.0 per)
			// Savings = 4.0 - (2 * 1.0) = 2.0
			expected: []domain.RoutingRecommendation{
				{
					TaskCategory:        "c1",
					CurrentModel:        "m2",
					RecommendedModel:    "m1",
					EstimatedSavingsUSD: 2.0,
					Reason:              "Routing this task to m1 instead of m2 will save $2.00 with zero latency penalty.",
				},
			},
		},
		{
			name: "savings too small",
			events: []domain.TokenEvent{
				// Model 1: 1 accepted, cost 1.0
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
				// Model 2: 1 accepted, cost 1.005 -> savings 0.005 (<= 0.01)
				{TaskCategory: "c1", ModelID: "m2", CostEstimateUSD: ptr(1.005), OutputStatus: domain.OutputAccepted},
			},
			expected: []domain.RoutingRecommendation{},
		},
		{
			name: "multiple categories",
			events: []domain.TokenEvent{
				// Cat c1 -> m1 (1.0 per) vs m2 (3.0 per) => savings 2.0
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: ptr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "m2", CostEstimateUSD: ptr(3.0), OutputStatus: domain.OutputAccepted},
				// Cat c2 -> m3 (2.0 per) vs m4 (5.0 per) => savings 3.0
				{TaskCategory: "c2", ModelID: "m3", CostEstimateUSD: ptr(2.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c2", ModelID: "m4", CostEstimateUSD: ptr(5.0), OutputStatus: domain.OutputAccepted},
			},
			expected: []domain.RoutingRecommendation{
				{
					TaskCategory:        "c1",
					CurrentModel:        "m2",
					RecommendedModel:    "m1",
					EstimatedSavingsUSD: 2.0,
					Reason:              "Routing this task to m1 instead of m2 will save $2.00 with zero latency penalty.",
				},
				{
					TaskCategory:        "c2",
					CurrentModel:        "m4",
					RecommendedModel:    "m3",
					EstimatedSavingsUSD: 3.0,
					Reason:              "Routing this task to m3 instead of m4 will save $3.00 with zero latency penalty.",
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

			// For simplicity we just check if expected matches exactly in an unordered way
			// We'll map them by Category for easy checking
			expectedMap := make(map[string]domain.RoutingRecommendation)
			for _, r := range tt.expected {
				expectedMap[r.TaskCategory] = r
			}

			for _, got := range recs {
				want, ok := expectedMap[got.TaskCategory]
				if !ok {
					t.Errorf("unexpected recommendation for category %s", got.TaskCategory)
					continue
				}
				if got.CurrentModel != want.CurrentModel {
					t.Errorf("expected current model %s, got %s", want.CurrentModel, got.CurrentModel)
				}
				if got.RecommendedModel != want.RecommendedModel {
					t.Errorf("expected recommended model %s, got %s", want.RecommendedModel, got.RecommendedModel)
				}
				if got.EstimatedSavingsUSD != want.EstimatedSavingsUSD {
					t.Errorf("expected savings %f, got %f", want.EstimatedSavingsUSD, got.EstimatedSavingsUSD)
				}
				if got.Reason != want.Reason {
					t.Errorf("expected reason %q, got %q", want.Reason, got.Reason)
				}
			}
		})
	}
}
