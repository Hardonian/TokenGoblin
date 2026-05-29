package analysis

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

const maxEvidenceChars = 1200

var sentenceSplitter = regexp.MustCompile(`[.!?]\s+`)

func Analyze(event domain.TokenEvent, analyzedAt time.Time) domain.OutputAnalysis {
	result := domain.OutputAnalysis{
		AnalysisID:      event.EventID + ":analysis",
		TenantID:        event.TenantID,
		EventID:         event.EventID,
		WorkerID:        event.WorkerID,
		AnalyzedAt:      analyzedAt.UTC(),
		EfficiencyScore: 100,
		GoblinScore:     100,
		Issues:          []domain.AnalysisIssue{},
		Recommendations: []string{},
		Evidence:        []domain.AnalysisEvidence{},
	}

	inputTokens := event.InputTokens
	if inputTokens == 0 {
		inputTokens = event.PromptTokens
	}
	outputTokens := event.OutputTokens
	if outputTokens == 0 {
		outputTokens = event.CompletionTokens
	}
	totalTokens := inputTokens + outputTokens
	if totalTokens == 0 {
		totalTokens = event.TotalTokens
	}
	if totalTokens > 0 {
		result.Evidence = append(result.Evidence, domain.AnalysisEvidence{
			Type:   "metric",
			Label:  "total_tokens",
			Metric: float64(totalTokens),
		})
	}

	if outputTokens > 1500 {
		addIssue(&result, "output_bloat", domain.SeverityMed, "Output token count is high for a single reviewed event.", fmt.Sprintf("%d output tokens", outputTokens), 16, "Cap output length and require concise structured sections.")
	}
	if inputTokens > 0 && outputTokens > inputTokens*2 {
		ratio := float64(outputTokens) / float64(inputTokens)
		addIssue(&result, "verbosity", domain.SeverityMed, "Output is more than twice the input size.", fmt.Sprintf("%.2fx output/input token ratio", ratio), 14, "Reduce verbosity and ask for only decision-useful details.")
	}
	if event.CachedTokens == 0 && inputTokens > 4000 {
		addIssue(&result, "duplicate_context_risk", domain.SeverityLow, "Large uncached input suggests repeated context may be inflating cost.", fmt.Sprintf("%d uncached input tokens", inputTokens), 8, "Use summaries for long history and remove duplicate context.")
	}
	if event.CostEstimateUSD != nil && *event.CostEstimateUSD > 1 {
		addIssue(&result, "cost_leak", domain.SeverityMed, "Single event has a high estimated cost.", fmt.Sprintf("$%.4f estimated", *event.CostEstimateUSD), 10, "Reserve expensive models for high-judgment work and route simple tasks cheaper.")
	}

	output := strings.TrimSpace(event.OutputExcerpt)
	if output == "" {
		result.Degraded = append(result.Degraded, domain.Issue{
			Code:    "insufficient_text_evidence",
			Message: "No output excerpt was provided; text-structure checks were skipped.",
			Field:   "output_excerpt",
		})
	} else {
		analyzeText(output, &result)
		result.Evidence = append(result.Evidence, domain.AnalysisEvidence{
			Type:  "excerpt",
			Label: "output_excerpt",
			Value: truncate(output, 220),
		})
	}

	if strings.TrimSpace(event.PromptExcerpt) == "" && strings.TrimSpace(event.PromptReference) == "" {
		result.Degraded = append(result.Degraded, domain.Issue{
			Code:    "insufficient_prompt_evidence",
			Message: "No prompt excerpt or prompt reference was provided; prompt-constraint checks were skipped.",
			Field:   "prompt_excerpt",
		})
	} else if strings.TrimSpace(event.PromptExcerpt) != "" {
		analyzePrompt(event.PromptExcerpt, &result)
	}

	if toolCalls, ok := intTag(event.Tags, "tool_calls"); ok && toolCalls > 0 && totalTokens > 0 && totalTokens < 800 {
		addIssue(&result, "unnecessary_tool_use", domain.SeverityLow, "Tool use was recorded for a small event where deterministic routing should verify necessity.", fmt.Sprintf("%d tool calls, %d total tokens", toolCalls, totalTokens), 7, "Add validation gates before tool use and skip tools for simple tasks.")
	}

	result.GoblinScore = result.EfficiencyScore
	if result.EfficiencyScore < 0 {
		result.EfficiencyScore = 0
		result.GoblinScore = 0
	}
	result.Recommendations = dedupe(result.Recommendations)
	return result
}

func analyzeText(output string, result *domain.OutputAnalysis) {
	lower := strings.ToLower(output)
	if len(output) > maxEvidenceChars {
		addIssue(result, "verbosity", domain.SeverityLow, "Output excerpt is long and may hide the decision signal.", fmt.Sprintf("%d characters in excerpt", len(output)), 6, "Reduce verbosity and make the first section the answer.")
	}
	if strings.Count(output, "\n") < 2 && len(output) > 500 {
		addIssue(result, "weak_structure", domain.SeverityMed, "Long output has little visible structure.", "few line breaks in long output", 12, "Require structured output with short labeled sections.")
	}
	if repeatedSentence(output) {
		addIssue(result, "repetition", domain.SeverityMed, "Repeated sentences or clauses were detected.", "duplicate sentence fragment", 14, "Remove duplicate context and repeated conclusions.")
	}
	if !strings.Contains(lower, "verify") && !strings.Contains(lower, "validated") && !strings.Contains(lower, "evidence") && !strings.Contains(lower, "checked") {
		addIssue(result, "missing_verification", domain.SeverityLow, "Output does not mention verification, evidence, or checks.", "no verification marker in output excerpt", 7, "Require verification notes or evidence IDs for review-heavy tasks.")
	}
}

func analyzePrompt(prompt string, result *domain.OutputAnalysis) {
	lower := strings.ToLower(prompt)
	if !strings.Contains(lower, "format") && !strings.Contains(lower, "schema") && !strings.Contains(lower, "json") && !strings.Contains(lower, "bullets") {
		addIssue(result, "poor_constraints", domain.SeverityLow, "Prompt excerpt does not show explicit output structure constraints.", "no visible format/schema constraint", 6, "Improve prompt templates with structure, limits, and required evidence.")
	}
	if !strings.Contains(lower, "limit") && !strings.Contains(lower, "concise") && !strings.Contains(lower, "brief") {
		addIssue(result, "missing_length_cap", domain.SeverityLow, "Prompt excerpt does not show a length constraint.", "no visible length cap", 5, "Cap output length for routine tasks.")
	}
}

func addIssue(a *domain.OutputAnalysis, code string, severity domain.Severity, message string, evidence string, penalty int, recommendation string) {
	a.Issues = append(a.Issues, domain.AnalysisIssue{
		Code:     code,
		Severity: severity,
		Message:  message,
		Evidence: evidence,
	})
	a.Recommendations = append(a.Recommendations, recommendation)
	a.Evidence = append(a.Evidence, domain.AnalysisEvidence{
		Type:  "rule",
		Label: code,
		Value: evidence,
	})
	a.EfficiencyScore -= penalty
}

func repeatedSentence(output string) bool {
	seen := map[string]bool{}
	for _, raw := range sentenceSplitter.Split(output, -1) {
		normalized := strings.Join(strings.Fields(strings.ToLower(raw)), " ")
		if len(normalized) < 32 {
			continue
		}
		if seen[normalized] {
			return true
		}
		seen[normalized] = true
	}
	return false
}

func intTag(tags map[string]string, key string) (int, bool) {
	if tags == nil {
		return 0, false
	}
	value, ok := tags[key]
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
