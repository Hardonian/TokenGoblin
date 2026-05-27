package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	StatusPriced   = "priced"
	StatusDegraded = "degraded"
)

type Registry struct {
	prices      map[string]domain.PricePoint
	diagnostics []domain.Issue
	redis       *redis.Client
}

type RegistryConfig struct {
	DisableDefaults bool
	PricingJSON     string
	RedisAddr       string
}

func ConfigFromEnv() RegistryConfig {
	return RegistryConfig{
		DisableDefaults: os.Getenv("TG_DISABLE_DEFAULT_PRICING") == "1",
		PricingJSON:     os.Getenv("TG_PRICING_TABLE_JSON"),
		RedisAddr:       os.Getenv("TG_REDIS_ADDR"),
	}
}

func LoadRegistry(ctx context.Context, config RegistryConfig) Registry {
	registry := Registry{prices: map[string]domain.PricePoint{}}
	
	if config.RedisAddr != "" {
		registry.setupRedis(ctx, config.RedisAddr)
	}

	if !config.DisableDefaults {
		for _, point := range DefaultPrices() {
			registry.prices[key(point.Provider, point.ModelID)] = point
		}
	}

	if strings.TrimSpace(config.PricingJSON) == "" {
		return registry
	}

	var raw map[string]priceJSON
	if err := json.Unmarshal([]byte(config.PricingJSON), &raw); err != nil {
		registry.diagnostics = append(registry.diagnostics, domain.Issue{
			Code:    "pricing_env_invalid",
			Message: "Pricing override JSON could not be parsed; using available registry entries.",
		})
		return registry
	}

	for rawKey, value := range raw {
		provider, modelID, ok := splitKey(rawKey)
		if !ok || value.InputPerMillion < 0 || value.OutputPerMillion < 0 {
			registry.diagnostics = append(registry.diagnostics, domain.Issue{
				Code:    "pricing_entry_invalid",
				Message: fmt.Sprintf("Pricing entry %q was ignored.", rawKey),
			})
			continue
		}
		currency := strings.ToUpper(strings.TrimSpace(value.Currency))
		if currency == "" {
			currency = "USD"
		}
		registry.prices[key(provider, modelID)] = domain.PricePoint{
			Provider:                  provider,
			ModelID:                   modelID,
			EffectiveFrom:             time.Unix(0, 0).UTC(),
			InputCostPerMillion:       value.InputPerMillion,
			OutputCostPerMillion:      value.OutputPerMillion,
			CachedInputCostPerMillion: value.CachedInputPerMillion,
			Currency:                  currency,
			Source:                    "env",
		}
	}

	return registry
}

func (r *Registry) setupRedis(ctx context.Context, addr string) {
	r.redis = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := r.redis.Ping(ctx).Err(); err != nil {
		r.diagnostics = append(r.diagnostics, domain.Issue{
			Code:    "redis_unavailable",
			Message: "Redis cache is unavailable; falling back to memory/defaults.",
		})
		r.redis = nil
	}
}

func (r Registry) Diagnostics() []domain.Issue {
	return append([]domain.Issue(nil), r.diagnostics...)
}

type OverrideFetcher interface {
	GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error)
}

func (r Registry) Calculate(ctx context.Context, event domain.TokenEvent, fetcher OverrideFetcher) domain.CostResult {
	var point domain.PricePoint
	var ok bool

	if fetcher != nil {
		if override, err := fetcher.GetPricingOverride(ctx, event.TenantID, event.Provider, event.ModelID); err == nil && override != nil {
			point = *override
			ok = true
		}
	}

	if !ok {
		point, ok = r.prices[key(event.Provider, event.ModelID)]
	}
	
	if !ok {
		return domain.CostResult{
			Status:        StatusDegraded,
			Currency:      "USD",
			DegradedCode:  "unknown_model_pricing",
			PricingSource: "none",
		}
	}
	if point.Currency != "USD" {
		return domain.CostResult{
			Status:        StatusDegraded,
			Currency:      "USD",
			DegradedCode:  "unsupported_pricing_currency",
			PricingSource: point.Source,
		}
	}

	cachedTokens := min(max(event.CachedTokens, 0), max(event.InputTokens, 0))
	uncachedInput := max(event.InputTokens-cachedTokens, 0)
	inputCost := float64(uncachedInput) * point.InputCostPerMillion / 1_000_000
	cachedCost := float64(cachedTokens) * point.CachedInputCostPerMillion / 1_000_000
	outputCost := float64(max(event.OutputTokens, 0)) * point.OutputCostPerMillion / 1_000_000
	total := roundUSD(inputCost + cachedCost + outputCost)

	return domain.CostResult{
		Status:          StatusPriced,
		CostEstimateUSD: &total,
		Currency:        "USD",
		PricingSource:   point.Source,
	}
}

type priceJSON struct {
	InputPerMillion       float64 `json:"inputPerMillion"`
	OutputPerMillion      float64 `json:"outputPerMillion"`
	CachedInputPerMillion float64 `json:"cachedInputPerMillion"`
	Currency              string  `json:"currency"`
}

func DefaultPrices() []domain.PricePoint {
	epoch := time.Unix(0, 0).UTC()
	return []domain.PricePoint{
		{
			Provider:                  "openai",
			ModelID:                   "gpt-4o-mini",
			EffectiveFrom:             epoch,
			InputCostPerMillion:       0.15,
			OutputCostPerMillion:      0.60,
			CachedInputCostPerMillion: 0.075,
			Currency:                  "USD",
			Source:                    "default",
		},
		{
			Provider:                  "openai",
			ModelID:                   "gpt-4o",
			EffectiveFrom:             epoch,
			InputCostPerMillion:       5.00,
			OutputCostPerMillion:      15.00,
			CachedInputCostPerMillion: 2.50,
			Currency:                  "USD",
			Source:                    "default",
		},
		{
			Provider:                  "anthropic",
			ModelID:                   "claude-3-5-haiku",
			EffectiveFrom:             epoch,
			InputCostPerMillion:       0.80,
			OutputCostPerMillion:      4.00,
			CachedInputCostPerMillion: 0.40,
			Currency:                  "USD",
			Source:                    "default",
		},
		{
			Provider:                  "anthropic",
			ModelID:                   "claude-3-5-sonnet",
			EffectiveFrom:             epoch,
			InputCostPerMillion:       3.00,
			OutputCostPerMillion:      15.00,
			CachedInputCostPerMillion: 1.50,
			Currency:                  "USD",
			Source:                    "default",
		},
		{
			Provider:                  "demo",
			ModelID:                   "efficient-model",
			EffectiveFrom:             epoch,
			InputCostPerMillion:       0.10,
			OutputCostPerMillion:      0.25,
			CachedInputCostPerMillion: 0.05,
			Currency:                  "USD",
			Source:                    "default",
		},
		{
			Provider:                  "demo",
			ModelID:                   "expensive-model",
			EffectiveFrom:             epoch,
			InputCostPerMillion:       12.00,
			OutputCostPerMillion:      36.00,
			CachedInputCostPerMillion: 6.00,
			Currency:                  "USD",
			Source:                    "default",
		},
	}
}

func key(provider string, modelID string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + ":" + strings.ToLower(strings.TrimSpace(modelID))
}

func splitKey(raw string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return strings.ToLower(strings.TrimSpace(parts[0])), strings.ToLower(strings.TrimSpace(parts[1])), true
}

func roundUSD(value float64) float64 {
	return math.Round(value*100_000_000) / 100_000_000
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
