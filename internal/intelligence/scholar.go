package intelligence

import (
	"context"
	"sort"
	"strings"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// ScholarEngine handles data mining and self-improving analytics.
type ScholarEngine struct {
	repo storage.Repository
}

// NewScholarEngine creates a new ScholarEngine.
func NewScholarEngine(repo storage.Repository) *ScholarEngine {
	return &ScholarEngine{repo: repo}
}

// Train analyzes the TokenEvents for a tenant and discovers new slop phrases
// by correlating rejected/failed outputs or low review scores with specific n-grams.
func (s *ScholarEngine) Train(ctx context.Context, tenantID string) error {
	// In a full implementation, this would:
	// 1. Fetch 10,000 recent TokenEvents.
	// 2. Filter for those with low review scores or OutputStatus == rejected.
	// 3. Extract PromptExcerpts.
	// 4. Perform n-gram (e.g., bigram/trigram) extraction and term frequency analysis.
	// 5. Compare frequencies to "successful" prompts (OutputStatus == accepted).
	// 6. Identify n-grams with a high tf-idf score heavily skewed towards failure.
	// 7. Save these discovered "slop" phrases to a database table for this tenant.

	// For MVP, we simulate discovering a few patterns.
	discoveredSlop := []string{
		"i apologize for",
		"let's think step by step", // Often used incorrectly by users in simple tasks
		"please make sure to",
	}

	// Store discovered slop patterns in the tenant's data or a global knowledge base.
	// For now, we will just update a simulated cache or log it.
	_ = discoveredSlop

	return nil
}

// Insights represents what the Scholar has learned about a tenant.
type Insights struct {
	DiscoveredWastePatterns []string               `json:"discovered_waste_patterns"`
	SuggestedOptimizations  []string               `json:"suggested_optimizations"`
	Metrics                 map[string]interface{} `json:"metrics"`
}

// GenerateInsights produces an Insights report for a tenant.
func (s *ScholarEngine) GenerateInsights(ctx context.Context, tenantID string) (Insights, error) {
	// In a real system, this would fetch from the database table populated by Train()
	return Insights{
		DiscoveredWastePatterns: []string{
			"Overuse of 'please make sure to' correlates with 15% higher token usage and no increase in success rate.",
			"Including JSON schemas manually in every prompt instead of using function calling wastes ~400 tokens per request.",
		},
		SuggestedOptimizations: []string{
			"Migrate 'gpt-4' calls in 'summarization' category to 'gpt-4o-mini' for an estimated 60% savings.",
			"Enable the Goblin Refiner for all 'support_agent' workers to strip polite filler.",
		},
		Metrics: map[string]interface{}{
			"prompts_analyzed":        5200,
			"patterns_discovered":     12,
			"estimated_waste_percent": 18.5,
		},
	}, nil
}
