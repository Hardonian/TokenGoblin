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
		setup    func(t *testing.T)
		expected bool
	}{
		{
			name:     "no env vars set",
			setup:    func(t *testing.T) {},
			expected: false,
		},
		{
			name: "TG_ENV is production",
			setup: func(t *testing.T) {
				t.Setenv("TG_ENV", "production")
			},
			expected: true,
		},
		{
			name: "APP_ENV is production",
			setup: func(t *testing.T) {
				t.Setenv("APP_ENV", "production")
			},
			expected: true,
		},
		{
			name: "GO_ENV is production",
			setup: func(t *testing.T) {
				t.Setenv("GO_ENV", "production")
			},
			expected: true,
		},
		{
			name: "NODE_ENV is production",
			setup: func(t *testing.T) {
				t.Setenv("NODE_ENV", "production")
			},
			expected: true,
		},
		{
			name: "VERCEL_ENV is production",
			setup: func(t *testing.T) {
				t.Setenv("VERCEL_ENV", "production")
			},
			expected: true,
		},
		{
			name: "different casing",
			setup: func(t *testing.T) {
				t.Setenv("TG_ENV", "PrOdUcTiOn")
			},
			expected: true,
		},
		{
			name: "with spaces",
			setup: func(t *testing.T) {
				t.Setenv("TG_ENV", "  production  ")
			},
			expected: true,
		},
		{
			name: "non-production env",
			setup: func(t *testing.T) {
				t.Setenv("TG_ENV", "development")
			},
			expected: false,
		},
		{
			name: "multiple envs, one is production",
			setup: func(t *testing.T) {
				t.Setenv("APP_ENV", "staging")
				t.Setenv("NODE_ENV", "production")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := []string{"TG_ENV", "APP_ENV", "GO_ENV", "NODE_ENV", "VERCEL_ENV"}
			for _, k := range keys {
				t.Setenv(k, "")
			}

			tt.setup(t)

			assert.Equal(t, tt.expected, IsProduction())
		})
	}
}
