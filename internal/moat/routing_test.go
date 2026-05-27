package moat

import (
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func TestRecommendRoutes(t *testing.T) {
	costPtr := func(v float64) *float64 { return &v }

	tests := []struct {
		name     string
		events   []domain.TokenEvent
		wantRecs int
		check    func(t *testing.T, recs []domain.RoutingRecommendation)
	}{
		{
			name:     "empty events",
			events:   []domain.TokenEvent{},
			wantRecs: 0,
		},
		{
			name: "missing fields ignored",
			events: []domain.TokenEvent{
				{OutputStatus: domain.OutputAccepted, ModelID: "m1", CostEstimateUSD: costPtr(1.0)}, // missing category
				{TaskCategory: "c1", OutputStatus: domain.OutputAccepted, CostEstimateUSD: costPtr(1.0)}, // missing model
				{TaskCategory: "c1", OutputStatus: domain.OutputAccepted, ModelID: "m1"}, // missing cost
			},
			wantRecs: 0,
		},
		{
			name: "non-accepted status ignored",
			events: []domain.TokenEvent{
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: costPtr(1.0), OutputStatus: domain.OutputFailed},
				{TaskCategory: "c1", ModelID: "m2", CostEstimateUSD: costPtr(0.5), OutputStatus: domain.OutputRejected},
			},
			wantRecs: 0,
		},
		{
			name: "single model no recommendation",
			events: []domain.TokenEvent{
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: costPtr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "m1", CostEstimateUSD: costPtr(1.0), OutputStatus: domain.OutputSucceeded},
			},
			wantRecs: 0,
		},
		{
			name: "savings under threshold",
			events: []domain.TokenEvent{
				{TaskCategory: "c1", ModelID: "cheap", CostEstimateUSD: costPtr(1.000), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "expensive", CostEstimateUSD: costPtr(1.005), OutputStatus: domain.OutputAccepted},
			},
			wantRecs: 0,
		},
		{
			name: "clear savings",
			events: []domain.TokenEvent{
				// Cheap model: 2 accepted, total cost 1.0 (0.5 each)
				{TaskCategory: "c1", ModelID: "cheap", CostEstimateUSD: costPtr(0.5), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "cheap", CostEstimateUSD: costPtr(0.5), OutputStatus: domain.OutputAccepted},
				// Expensive model: 2 accepted, total cost 3.0 (1.5 each)
				{TaskCategory: "c1", ModelID: "expensive", CostEstimateUSD: costPtr(1.5), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "expensive", CostEstimateUSD: costPtr(1.5), OutputStatus: domain.OutputAccepted},
			},
			wantRecs: 1,
			check: func(t *testing.T, recs []domain.RoutingRecommendation) {
				rec := recs[0]
				if rec.TaskCategory != "c1" {
					t.Errorf("got category %q, want c1", rec.TaskCategory)
				}
				if rec.CurrentModel != "expensive" {
					t.Errorf("got current model %q, want expensive", rec.CurrentModel)
				}
				if rec.RecommendedModel != "cheap" {
					t.Errorf("got recommended model %q, want cheap", rec.RecommendedModel)
				}
				// worstTotalCost = 3.0
				// projectedCost = 2 * 0.5 = 1.0
				// savings = 2.0
				if rec.EstimatedSavingsUSD != 2.0 {
					t.Errorf("got savings %f, want 2.0", rec.EstimatedSavingsUSD)
				}
			},
		},
		{
			name: "zero accepted count ignored for model stats",
			events: []domain.TokenEvent{
				// Cheap model gets no accepted outputs, only failed
				{TaskCategory: "c1", ModelID: "cheap", CostEstimateUSD: costPtr(0.5), OutputStatus: domain.OutputFailed},
				// Expensive model gets accepted
				{TaskCategory: "c1", ModelID: "expensive", CostEstimateUSD: costPtr(1.5), OutputStatus: domain.OutputAccepted},
			},
			wantRecs: 0,
		},
		{
			name: "multiple categories",
			events: []domain.TokenEvent{
				{TaskCategory: "c1", ModelID: "cheap1", CostEstimateUSD: costPtr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c1", ModelID: "expensive1", CostEstimateUSD: costPtr(2.0), OutputStatus: domain.OutputAccepted},

				{TaskCategory: "c2", ModelID: "cheap2", CostEstimateUSD: costPtr(1.0), OutputStatus: domain.OutputAccepted},
				{TaskCategory: "c2", ModelID: "expensive2", CostEstimateUSD: costPtr(2.0), OutputStatus: domain.OutputAccepted},
			},
			wantRecs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := RecommendRoutes(tt.events)
			if len(recs) != tt.wantRecs {
				t.Fatalf("got %d recommendations, want %d", len(recs), tt.wantRecs)
			}
			// Note: The function guarantees an initialized slice `recs = []domain.RoutingRecommendation{}` if no recs,
			// which means `recs != nil` should always hold true based on the final block of code in `routing.go`.
			// See routing.go lines 90-93:
			// if recs == nil {
			// 	recs = []domain.RoutingRecommendation{}
			// }
			if tt.wantRecs == 0 && recs == nil {
				t.Errorf("expected empty slice, got nil")
			}
			if tt.check != nil && len(recs) > 0 {
				tt.check(t, recs)
			}
		})
	}
}
