package domain

import "time"

// PricePoint defines the cost of a model at a specific point in time.
type PricePoint struct {
	Provider                  string    `json:"provider"`
	ModelID                   string    `json:"model_id"`
	EffectiveFrom             time.Time `json:"effective_from"`
	EffectiveTo               time.Time `json:"effective_to,omitempty"`
	InputCostPerMillion       float64   `json:"input_cost_per_million"`
	OutputCostPerMillion      float64   `json:"output_cost_per_million"`
	CachedInputCostPerMillion float64   `json:"cached_input_cost_per_million,omitempty"`
	Currency                  string    `json:"currency"`
	Source                    string    `json:"source"`
}

type CostResult struct {
	Status          string   `json:"status"`
	CostEstimateUSD *float64 `json:"cost_estimate_usd,omitempty"`
	Currency        string   `json:"currency"`
	DegradedCode    string   `json:"degraded_code,omitempty"`
	PricingSource   string   `json:"pricing_source"`
}
