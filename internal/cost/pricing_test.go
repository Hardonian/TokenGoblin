package cost

import (
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func TestCalculateKnownModelUsesInternalPricing(t *testing.T) {
	registry := LoadRegistry(RegistryConfig{})
	result := registry.Calculate(domain.TokenEvent{
		Provider:     "demo",
		ModelID:      "efficient-model",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		CachedTokens: 100_000,
	})

	if result.Status != StatusPriced {
		t.Fatalf("expected priced result, got %s", result.Status)
	}
	if result.CostEstimateUSD == nil {
		t.Fatal("expected cost estimate")
	}
	if *result.CostEstimateUSD != 0.345 {
		t.Fatalf("expected 0.345 USD, got %.8f", *result.CostEstimateUSD)
	}
}

func TestCalculateUnknownModelDegrades(t *testing.T) {
	registry := LoadRegistry(RegistryConfig{DisableDefaults: true})
	result := registry.Calculate(domain.TokenEvent{
		Provider:     "unknown",
		ModelID:      "unknown-model",
		InputTokens:  100,
		OutputTokens: 50,
	})

	if result.Status != StatusDegraded {
		t.Fatalf("expected degraded result, got %s", result.Status)
	}
	if result.DegradedCode != "unknown_model_pricing" {
		t.Fatalf("expected unknown pricing code, got %q", result.DegradedCode)
	}
	if result.CostEstimateUSD != nil {
		t.Fatal("unknown pricing must not produce trusted cost")
	}
}

func TestPricingEnvOverride(t *testing.T) {
	registry := LoadRegistry(RegistryConfig{
		DisableDefaults: true,
		PricingJSON:     `{"custom:model-a":{"inputPerMillion":2,"outputPerMillion":4,"currency":"USD"}}`,
	})
	result := registry.Calculate(domain.TokenEvent{
		Provider:     "custom",
		ModelID:      "model-a",
		InputTokens:  500_000,
		OutputTokens: 250_000,
	})

	if result.CostEstimateUSD == nil || *result.CostEstimateUSD != 2 {
		t.Fatalf("expected override cost 2 USD, got %#v", result.CostEstimateUSD)
	}
}
