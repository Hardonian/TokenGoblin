package intelligence

import (
	"regexp"
	"strings"
)

// RefinePrompt analyzes a prompt, removes conversational slop, trims excess whitespace,
// and minifies any JSON structures to reduce token count before sending to an LLM.
func RefinePrompt(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	refined := input

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
		// Dynamic patterns simulated from Scholar Engine
		"i apologize for",
		"let's think step by step",
		"please make sure to",
	}

	for _, phrase := range slopPhrases {
		// Case insensitive replacement using regex for word boundaries
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(phrase) + `\b[ \t]*`)
		refined = re.ReplaceAllString(refined, "")
	}

	// 2. Remove filler adjectives/adverbs
	fillerAdjectives := []string{
		"very", "extremely", "really", "basically", "actually", "literally", "simply",
	}
	for _, phrase := range fillerAdjectives {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(phrase) + `\b[ \t]*`)
		refined = re.ReplaceAllString(refined, "")
	}

	// 3. Compact multiple spaces, tabs, and line breaks
	reSpaces := regexp.MustCompile(`[ \t]+`)
	refined = reSpaces.ReplaceAllString(refined, " ")

	reBreaks := regexp.MustCompile(`\n\s*\n`)
	refined = reBreaks.ReplaceAllString(refined, "\n")

	// 4. (Optional) Basic JSON minification if JSON blocks are detected
	// A simple heuristic: if it looks like a JSON block, remove spaces around braces and brackets.
	refined = regexp.MustCompile(`\s*\{\s*`).ReplaceAllString(refined, "{")
	refined = regexp.MustCompile(`\s*\}\s*`).ReplaceAllString(refined, "}")
	refined = regexp.MustCompile(`\s*\[\s*`).ReplaceAllString(refined, "[")
	refined = regexp.MustCompile(`\s*\]\s*`).ReplaceAllString(refined, "]")
	refined = regexp.MustCompile(`\s*:\s*`).ReplaceAllString(refined, ":")

	// Final trim
	refined = strings.TrimSpace(refined)

	// Ensure we don't return an empty string if it was all slop (though unlikely)
	if refined == "" && len(input) > 0 {
		return input
	}

	return refined
}
