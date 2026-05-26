package domain

import "time"

// PricePoint defines the cost of a model at a specific point in time.
type PricePoint struct {
	ModelID             string    `json:"model_id"`
	EffectiveFrom       time.Time `json:"effective_from"`
	EffectiveTo         time.Time `json:"effective_to"`
	PromptCostPer1k     float64   `json:"prompt_cost_per_1k"`
	CompletionCostPer1k float64   `json:"completion_cost_per_1k"`
}
