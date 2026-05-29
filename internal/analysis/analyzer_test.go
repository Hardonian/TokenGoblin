package analysis

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func TestAnalyzeDetectsDeterministicWasteSignals(t *testing.T) {
	cost := 1.25
	result := Analyze(domain.TokenEvent{
		EventID:         "evt-1",
		TenantID:        "tenant-a",
		WorkerID:        "worker-a",
		InputTokens:     500,
		OutputTokens:    1800,
		CachedTokens:    0,
		CostEstimateUSD: &cost,
		PromptExcerpt:   "Write an answer.",
		OutputExcerpt:   "This is repeated and should be reviewed. This is repeated and should be reviewed.",
		Tags:            map[string]string{"tool_calls": "1"},
	}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if result.EfficiencyScore >= 100 {
		t.Fatalf("expected reduced score, got %d", result.EfficiencyScore)
	}
	if len(result.Issues) == 0 {
		t.Fatal("expected deterministic issues")
	}
	if !hasIssue(result.Issues, "output_bloat") || !hasIssue(result.Issues, "verbosity") {
		t.Fatalf("expected output bloat and verbosity issues, got %#v", result.Issues)
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}
}

func TestAnalyzeDegradesWhenTextEvidenceMissing(t *testing.T) {
	result := Analyze(domain.TokenEvent{
		EventID:      "evt-2",
		TenantID:     "tenant-a",
		WorkerID:     "worker-a",
		InputTokens:  100,
		OutputTokens: 50,
	}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if len(result.Degraded) == 0 {
		t.Fatal("expected degraded evidence state")
	}
	if result.EfficiencyScore != 100 {
		t.Fatalf("missing excerpts should not fabricate a lower score, got %d", result.EfficiencyScore)
	}
}

func hasIssue(issues []domain.AnalysisIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
