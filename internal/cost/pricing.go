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

	inputTokens := event.InputTokens
	if inputTokens == 0 && event.PromptExcerpt != "" {
		// Heuristic: ~4 chars per token for English
		inputTokens = len(event.PromptExcerpt) / 4
	}

	cachedTokens := min(max(event.CachedTokens, 0), max(inputTokens, 0))
	uncachedInput := max(inputTokens-cachedTokens, 0)
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
	p := func(provider, model string, input, output, cached float64) domain.PricePoint {
		return domain.PricePoint{
			Provider:                  provider,
			ModelID:                   model,
			EffectiveFrom:             epoch,
			InputCostPerMillion:       input,
			OutputCostPerMillion:      output,
			CachedInputCostPerMillion: cached,
			Currency:                  "USD",
			Source:                    "default",
		}
	}

	return []domain.PricePoint{
		// ═══════════════════════════════════════════════════
		// OpenAI
		// ═══════════════════════════════════════════════════
		p("openai", "gpt-4o", 2.50, 10.00, 1.25),
		p("openai", "gpt-4o-mini", 0.15, 0.60, 0.075),
		p("openai", "gpt-4.1", 2.00, 8.00, 0.50),
		p("openai", "gpt-4.1-mini", 0.40, 1.60, 0.10),
		p("openai", "gpt-4.1-nano", 0.10, 0.40, 0.025),
		p("openai", "o1", 15.00, 60.00, 7.50),
		p("openai", "o1-mini", 1.10, 4.40, 0.55),
		p("openai", "o1-pro", 150.00, 600.00, 0),
		p("openai", "o3", 10.00, 40.00, 2.50),
		p("openai", "o3-mini", 1.10, 4.40, 0.55),
		p("openai", "o4-mini", 1.10, 4.40, 0.275),
		p("openai", "gpt-4-turbo", 10.00, 30.00, 5.00),
		p("openai", "gpt-3.5-turbo", 0.50, 1.50, 0.25),

		// ═══════════════════════════════════════════════════
		// Anthropic
		// ═══════════════════════════════════════════════════
		p("anthropic", "claude-4-opus", 15.00, 75.00, 7.50),
		p("anthropic", "claude-4-sonnet", 3.00, 15.00, 1.50),
		p("anthropic", "claude-3-5-sonnet", 3.00, 15.00, 1.50),
		p("anthropic", "claude-3-5-haiku", 0.80, 4.00, 0.40),
		p("anthropic", "claude-3-opus", 15.00, 75.00, 7.50),
		p("anthropic", "claude-3-sonnet", 3.00, 15.00, 1.50),
		p("anthropic", "claude-3-haiku", 0.25, 1.25, 0.125),

		// ═══════════════════════════════════════════════════
		// Google — Gemini
		// ═══════════════════════════════════════════════════
		p("google", "gemini-2.5-pro", 1.25, 10.00, 0.3125),
		p("google", "gemini-2.5-flash", 0.15, 0.60, 0.0375),
		p("google", "gemini-2.0-flash", 0.10, 0.40, 0.025),
		p("google", "gemini-2.0-flash-lite", 0.075, 0.30, 0),
		p("google", "gemini-1.5-pro", 1.25, 5.00, 0.3125),
		p("google", "gemini-1.5-flash", 0.075, 0.30, 0.01875),
		// Alias: some callers use "gemini" as provider
		p("gemini", "gemini-2.5-pro", 1.25, 10.00, 0.3125),
		p("gemini", "gemini-2.5-flash", 0.15, 0.60, 0.0375),
		p("gemini", "gemini-2.0-flash", 0.10, 0.40, 0.025),

		// ═══════════════════════════════════════════════════
		// xAI — Grok
		// ═══════════════════════════════════════════════════
		p("xai", "grok-3", 3.00, 15.00, 0.75),
		p("xai", "grok-3-mini", 0.30, 0.50, 0.075),
		p("xai", "grok-2", 2.00, 10.00, 0),
		// Alias
		p("grok", "grok-3", 3.00, 15.00, 0.75),
		p("grok", "grok-3-mini", 0.30, 0.50, 0.075),

		// ═══════════════════════════════════════════════════
		// DeepSeek
		// ═══════════════════════════════════════════════════
		p("deepseek", "deepseek-chat", 0.14, 0.28, 0.014),
		p("deepseek", "deepseek-reasoner", 0.55, 2.19, 0.14),

		// ═══════════════════════════════════════════════════
		// Mistral
		// ═══════════════════════════════════════════════════
		p("mistral", "mistral-large", 2.00, 6.00, 0),
		p("mistral", "mistral-small", 0.10, 0.30, 0),
		p("mistral", "mistral-medium", 2.70, 8.10, 0),
		p("mistral", "codestral", 0.30, 0.90, 0),
		p("mistral", "mistral-nemo", 0.15, 0.15, 0),
		p("mistral", "pixtral-large", 2.00, 6.00, 0),

		// ═══════════════════════════════════════════════════
		// Meta — via hosted providers (OpenRouter/Together)
		// ═══════════════════════════════════════════════════
		p("meta", "llama-4-maverick", 0.20, 0.60, 0),
		p("meta", "llama-4-scout", 0.15, 0.40, 0),
		p("meta", "llama-3.3-70b", 0.40, 0.40, 0),
		p("meta", "llama-3.1-405b", 3.00, 3.00, 0),
		p("meta", "llama-3.1-70b", 0.40, 0.40, 0),
		p("meta", "llama-3.1-8b", 0.05, 0.05, 0),

		// ═══════════════════════════════════════════════════
		// OpenRouter — passthrough, common models
		// ═══════════════════════════════════════════════════
		p("openrouter", "openai/gpt-4o", 2.50, 10.00, 1.25),
		p("openrouter", "openai/gpt-4o-mini", 0.15, 0.60, 0.075),
		p("openrouter", "anthropic/claude-3-5-sonnet", 3.00, 15.00, 1.50),
		p("openrouter", "anthropic/claude-3-5-haiku", 0.80, 4.00, 0.40),
		p("openrouter", "google/gemini-2.5-flash", 0.15, 0.60, 0.0375),
		p("openrouter", "deepseek/deepseek-chat", 0.14, 0.28, 0.014),
		p("openrouter", "meta-llama/llama-3.1-70b", 0.40, 0.40, 0),

		// ═══════════════════════════════════════════════════
		// Azure OpenAI — same pricing as OpenAI
		// ═══════════════════════════════════════════════════
		p("azure", "gpt-4o", 2.50, 10.00, 1.25),
		p("azure", "gpt-4o-mini", 0.15, 0.60, 0.075),
		p("azure", "gpt-4.1", 2.00, 8.00, 0.50),
		p("azure", "gpt-4.1-mini", 0.40, 1.60, 0.10),
		p("azure", "gpt-4.1-nano", 0.10, 0.40, 0.025),
		p("azure-openai", "gpt-4o", 2.50, 10.00, 1.25),
		p("azure-openai", "gpt-4o-mini", 0.15, 0.60, 0.075),

		// ═══════════════════════════════════════════════════
		// AWS Bedrock
		// ═══════════════════════════════════════════════════
		p("bedrock", "anthropic.claude-3-5-sonnet", 3.00, 15.00, 1.50),
		p("bedrock", "anthropic.claude-3-5-haiku", 0.80, 4.00, 0.40),
		p("bedrock", "anthropic.claude-3-haiku", 0.25, 1.25, 0.125),
		p("bedrock", "amazon.nova-pro", 0.80, 3.20, 0),
		p("bedrock", "amazon.nova-lite", 0.06, 0.24, 0),
		p("bedrock", "amazon.nova-micro", 0.035, 0.14, 0),
		p("bedrock", "meta.llama-3.1-70b", 0.72, 0.72, 0),
		p("bedrock", "meta.llama-3.1-8b", 0.22, 0.22, 0),
		p("aws-bedrock", "anthropic.claude-3-5-sonnet", 3.00, 15.00, 1.50),

		// ═══════════════════════════════════════════════════
		// Google Cloud — Vertex AI (same as Gemini pricing)
		// ═══════════════════════════════════════════════════
		p("vertex", "gemini-2.5-pro", 1.25, 10.00, 0.3125),
		p("vertex", "gemini-2.5-flash", 0.15, 0.60, 0.0375),
		p("vertex", "gemini-2.0-flash", 0.10, 0.40, 0.025),
		p("vertex-ai", "gemini-2.5-pro", 1.25, 10.00, 0.3125),
		p("vertex-ai", "gemini-2.5-flash", 0.15, 0.60, 0.0375),

		// ═══════════════════════════════════════════════════
		// Cohere
		// ═══════════════════════════════════════════════════
		p("cohere", "command-r-plus", 2.50, 10.00, 0),
		p("cohere", "command-r", 0.15, 0.60, 0),
		p("cohere", "command-light", 0.30, 0.60, 0),

		// ═══════════════════════════════════════════════════
		// Ollama / LM Studio / Local — zero cost
		// ═══════════════════════════════════════════════════
		p("ollama", "llama3.1", 0, 0, 0),
		p("ollama", "llama3", 0, 0, 0),
		p("ollama", "mistral", 0, 0, 0),
		p("ollama", "codellama", 0, 0, 0),
		p("ollama", "gemma2", 0, 0, 0),
		p("ollama", "phi3", 0, 0, 0),
		p("ollama", "qwen2.5", 0, 0, 0),
		p("lmstudio", "local", 0, 0, 0),
		p("local", "local", 0, 0, 0),

		// ═══════════════════════════════════════════════════
		// Demo / Test
		// ═══════════════════════════════════════════════════
		p("demo", "efficient-model", 0.10, 0.25, 0.05),
		p("demo", "expensive-model", 12.00, 36.00, 6.00),
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
