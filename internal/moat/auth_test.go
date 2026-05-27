package moat

import (
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "standard valid token",
			header:   "Bearer mytoken123",
			expected: "mytoken123",
		},
		{
			name:     "lowercase bearer",
			header:   "bearer mytoken123",
			expected: "mytoken123",
		},
		{
			name:     "uppercase bearer",
			header:   "BEARER mytoken123",
			expected: "mytoken123",
		},
		{
			name:     "mixed case bearer",
			header:   "bEaReR mytoken123",
			expected: "mytoken123",
		},
		{
			name:     "wrong scheme",
			header:   "Basic mytoken123",
			expected: "",
		},
		{
			name:     "no space",
			header:   "Bearermytoken123",
			expected: "",
		},
		{
			name:     "only scheme and space",
			header:   "Bearer ",
			expected: "",
		},
		{
			name:     "multiple spaces",
			header:   "Bearer  mytoken123 ",
			expected: " mytoken123 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBearerToken(tt.header)
			if result != tt.expected {
				t.Errorf("ExtractBearerToken(%q) = %q; want %q", tt.header, result, tt.expected)
			}
		})
	}
}
