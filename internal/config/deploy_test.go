package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestIsProduction(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envVal   string
		expected bool
	}{
		{"default behavior no env", "", "", false},
		{"TG_ENV production", "TG_ENV", "production", true},
		{"APP_ENV production uppercase", "APP_ENV", "PRODUCTION", true},
		{"GO_ENV production padded", "GO_ENV", "  production  ", true},
		{"NODE_ENV mixed case", "NODE_ENV", "PrOdUcTiOn", true},
		{"VERCEL_ENV production", "VERCEL_ENV", "production", true},
		{"TG_ENV development", "TG_ENV", "development", false},
		{"unsupported key production", "UNSUPPORTED_ENV", "production", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear relevant env vars to isolate test case
			for _, key := range []string{"TG_ENV", "APP_ENV", "GO_ENV", "NODE_ENV", "VERCEL_ENV"} {
				t.Setenv(key, "")
			}
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envVal)
			}
			assert.Equal(t, tc.expected, IsProduction())
		})
	}
}
