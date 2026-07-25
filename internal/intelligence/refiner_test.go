package intelligence

import (
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestRefinePrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		profile  *domain.TuningProfile
		expected string
	}{
		{
			name:     "empty input",
			input:    "   \t\n  ",
			profile:  nil,
			expected: "",
		},
		{
			name:     "fast path 0.0 aggressiveness",
			input:    " please can you help me  ",
			profile:  &domain.TuningProfile{Aggressiveness: 0.0},
			expected: "please can you help me",
		},
		{
			name:     "default profile nil removes slop and extra adjectives",
			input:    "please can you really help me?",
			profile:  nil,
			expected: "help me?",
		},
		{
			name:     "low aggressiveness keeps extra adjectives",
			input:    "please can you really help me?",
			profile:  &domain.TuningProfile{Aggressiveness: 0.4},
			expected: "really help me?",
		},
		{
			name:     "high aggressiveness removes extra adjectives",
			input:    "please can you really help me?",
			profile:  &domain.TuningProfile{Aggressiveness: 0.6},
			expected: "help me?",
		},
		{
			name:     "ignored keywords are kept",
			input:    "please can you really help me?",
			profile:  &domain.TuningProfile{Aggressiveness: 0.6, IgnoredKeywords: []string{"please"}},
			expected: "please help me?",
		},
		{
			name:     "compacts spaces",
			input:    "word1    word2\t\tword3",
			profile:  &domain.TuningProfile{Aggressiveness: 0.4},
			expected: "word1 word2 word3",
		},
		{
			name:     "compacts line breaks",
			input:    "line1\n \n\nline2",
			profile:  &domain.TuningProfile{Aggressiveness: 0.4},
			expected: "line1\nline2",
		},
		{
			name:     "high aggressiveness minifies JSON",
			input:    "  { \n \"key\" : \n [ 1 , 2 ] }  ",
			profile:  &domain.TuningProfile{Aggressiveness: 0.9},
			expected: "{\"key\":[1 , 2]}",
		},
		{
			name:     "low aggressiveness does not minify JSON",
			input:    " { \n \"key\" : \n [ 1 , 2 ] } ",
			profile:  &domain.TuningProfile{Aggressiveness: 0.4},
			expected: "{ \n \"key\" : \n [ 1 , 2 ] }",
		},
		{
			name:     "case insensitive slop removal",
			input:    "PLEASE Thank You As An AI Language Model help me",
			profile:  nil,
			expected: "help me",
		},
		{
			name:     "fallback when entirely slop",
			input:    "please thank you",
			profile:  nil,
			expected: "please thank you",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RefinePrompt(tt.input, tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
}
