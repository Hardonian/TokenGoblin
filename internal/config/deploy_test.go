package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsProduction(t *testing.T) {
	envKeys := []string{"TG_ENV", "APP_ENV", "GO_ENV", "NODE_ENV", "VERCEL_ENV"}

	tests := []struct {
		name     string
		envKey   string
		envVal   string
		expected bool
	}{
		{
			name:     "no env set",
			expected: false,
		},
		{
			name:     "TG_ENV is production",
			envKey:   "TG_ENV",
			envVal:   "production",
			expected: true,
		},
		{
			name:     "NODE_ENV is production",
			envKey:   "NODE_ENV",
			envVal:   "production",
			expected: true,
		},
		{
			name:     "uppercase PRODUCTION",
			envKey:   "APP_ENV",
			envVal:   "PRODUCTION",
			expected: true,
		},
		{
			name:     "mixed case PrOdUcTiOn",
			envKey:   "GO_ENV",
			envVal:   "PrOdUcTiOn",
			expected: true,
		},
		{
			name:     "with surrounding spaces",
			envKey:   "VERCEL_ENV",
			envVal:   "  production  ",
			expected: true,
		},
		{
			name:     "development env",
			envKey:   "TG_ENV",
			envVal:   "development",
			expected: false,
		},
		{
			name:     "empty string env",
			envKey:   "TG_ENV",
			envVal:   "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range envKeys {
				t.Setenv(k, "")
			}

			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envVal)
			}

			assert.Equal(t, tc.expected, IsProduction())
		})
	}
}

func TestValidateServerEnvRequiresProductionBoundaries(t *testing.T) {
	t.Setenv("TG_ENV", "production")
	t.Setenv("TG_DB_DSN", "")
	t.Setenv("TG_INTERNAL_WEBHOOK_SECRET", "")

	if err := ValidateServerEnv(); err == nil {
		t.Fatal("expected production validation to fail without database and internal webhook secret")
	}
}

func TestValidateServerEnvAllowsLocalDemoMode(t *testing.T) {
	t.Setenv("TG_ENV", "development")
	t.Setenv("TG_ALLOW_DEMO_AUTH", "1")

	if err := ValidateServerEnv(); err != nil {
		t.Fatalf("expected local demo mode to pass: %v", err)
	}
}
