package cost

import (
	"context"
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCalculateKnownModelUsesInternalPricing(t *testing.T) {
	registry := LoadRegistry(context.Background(), RegistryConfig{})
	result := registry.Calculate(context.Background(), domain.TokenEvent{
		Provider:     "demo",
		ModelID:      "efficient-model",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		CachedTokens: 100_000,
	}, nil)

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
	registry := LoadRegistry(context.Background(), RegistryConfig{DisableDefaults: true})
	result := registry.Calculate(context.Background(), domain.TokenEvent{
		Provider:     "unknown",
		ModelID:      "unknown-model",
		InputTokens:  100,
		OutputTokens: 50,
	}, nil)

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
	registry := LoadRegistry(context.Background(), RegistryConfig{
		DisableDefaults: true,
		PricingJSON:     `{"custom:model-a":{"inputPerMillion":2,"outputPerMillion":4,"currency":"USD"}}`,
	})
	result := registry.Calculate(context.Background(), domain.TokenEvent{
		Provider:     "custom",
		ModelID:      "model-a",
		InputTokens:  500_000,
		OutputTokens: 250_000,
	}, nil)

	if result.CostEstimateUSD == nil || *result.CostEstimateUSD != 2 {
		t.Fatalf("expected override cost 2 USD, got %#v", result.CostEstimateUSD)
	}
}

func TestSplitKey(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		wantP1 string
		wantP2 string
		wantOk bool
	}{
		{
			name:   "standard case",
			raw:    "provider:model",
			wantP1: "provider",
			wantP2: "model",
			wantOk: true,
		},
		{
			name:   "with whitespace",
			raw:    "  provider  :  model  ",
			wantP1: "provider",
			wantP2: "model",
			wantOk: true,
		},
		{
			name:   "with uppercase characters",
			raw:    "ProViDer:MoDel",
			wantP1: "provider",
			wantP2: "model",
			wantOk: true,
		},
		{
			name:   "extra colons",
			raw:    "provider:model:version",
			wantP1: "provider",
			wantP2: "model:version",
			wantOk: true,
		},
		{
			name:   "missing colon",
			raw:    "providermodel",
			wantP1: "",
			wantP2: "",
			wantOk: false,
		},
		{
			name:   "empty first part",
			raw:    ":model",
			wantP1: "",
			wantP2: "",
			wantOk: false,
		},
		{
			name:   "empty second part",
			raw:    "provider:",
			wantP1: "",
			wantP2: "",
			wantOk: false,
		},
		{
			name:   "whitespace only parts",
			raw:    "   :   ",
			wantP1: "",
			wantP2: "",
			wantOk: false,
		},
		{
			name:   "empty string",
			raw:    "",
			wantP1: "",
			wantP2: "",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p1, p2, ok := splitKey(tt.raw)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantP1, p1)
			assert.Equal(t, tt.wantP2, p2)
		})
	}
}
