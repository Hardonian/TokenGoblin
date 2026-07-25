package intelligence

import (
	"regexp"
	"strings"
	"sync"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

var (
	reSpaces = regexp.MustCompile(`[ \t]+`)
	reBreaks = regexp.MustCompile(`\n\s*\n`)

	reJSONOpenBrace  = regexp.MustCompile(`\s*\{\s*`)
	reJSONCloseBrace = regexp.MustCompile(`\s*\}\s*`)
	reJSONOpenBrack  = regexp.MustCompile(`\s*\[\s*`)
	reJSONCloseBrack = regexp.MustCompile(`\s*\]\s*`)
	reJSONColon      = regexp.MustCompile(`\s*:\s*`)

	slopRegexesMu sync.RWMutex
	slopRegexes   = make(map[string]*regexp.Regexp)
)

func getSlopRegex(phrase string) *regexp.Regexp {
	slopRegexesMu.RLock()
	re, ok := slopRegexes[phrase]
	slopRegexesMu.RUnlock()
	if ok {
		return re
	}

	slopRegexesMu.Lock()
	defer slopRegexesMu.Unlock()

	// Double check
	if re, ok := slopRegexes[phrase]; ok {
		return re
	}

	re = regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(phrase) + `\b[ \t]*`)
	slopRegexes[phrase] = re
	return re
}

// RefinePrompt analyzes a prompt, removes conversational slop, trims excess whitespace,
// and minifies any JSON structures to reduce token count before sending to an LLM.
// It respects the TuningProfile's aggressiveness and ignored keywords.
func RefinePrompt(input string, profile *domain.TuningProfile) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	refined := input

	// Fast path for 0.0 aggressiveness
	if profile != nil && profile.Aggressiveness == 0.0 {
		return strings.TrimSpace(input)
	}

	// 1. Remove polite filler ("slop")
	slopPhrases := []string{
		"please",
		"could you",
		"kindly",
		"if you don't mind",
		"i would like you to",
		"can you",
		"thank you",
		"thanks in advance",
		"as an ai language model",
		"as an ai",
		"i am looking for",
		"i need you to",
		"it would be great if",
		"i apologize for",
		"let's think step by step",
		"please make sure to",
	}

	// Remove filler adjectives/adverbs if aggressiveness > 0.5
	if profile == nil || profile.Aggressiveness > 0.5 {
		slopPhrases = append(slopPhrases, "very", "extremely", "really", "basically", "actually", "literally", "simply")
	}

	for _, phrase := range slopPhrases {
		// Skip ignored keywords
		if isIgnored(phrase, profile) {
			continue
		}

		// Case insensitive replacement using regex for word boundaries
		re := getSlopRegex(phrase)
		refined = re.ReplaceAllString(refined, "")
	}

	// 3. Compact multiple spaces, tabs, and line breaks
	refined = reSpaces.ReplaceAllString(refined, " ")
	refined = reBreaks.ReplaceAllString(refined, "\n")

	// 4. JSON minification if aggressiveness is high enough
	if profile == nil || profile.Aggressiveness >= 0.8 {
		refined = reJSONOpenBrace.ReplaceAllString(refined, "{")
		refined = reJSONCloseBrace.ReplaceAllString(refined, "}")
		refined = reJSONOpenBrack.ReplaceAllString(refined, "[")
		refined = reJSONCloseBrack.ReplaceAllString(refined, "]")
		refined = reJSONColon.ReplaceAllString(refined, ":")
	}

	// Final trim
	refined = strings.TrimSpace(refined)

	if refined == "" && len(input) > 0 {
		return input
	}

	return refined
}

func isIgnored(phrase string, profile *domain.TuningProfile) bool {
	if profile == nil {
		return false
	}
	for _, ign := range profile.IgnoredKeywords {
		if strings.EqualFold(phrase, ign) {
			return true
		}
	}
	return false
}
