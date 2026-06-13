package config

import "testing"

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
		{
			name:     "no env vars set",
			envKey:   "",
			envVal:   "",
			expected: false,
		},
		{
			name:     "TG_ENV set to production",
			envKey:   "TG_ENV",
			envVal:   "production",
			expected: true,
		},
		{
			name:     "APP_ENV set to production",
			envKey:   "APP_ENV",
			envVal:   "production",
			expected: true,
		},
		{
			name:     "GO_ENV set to production",
			envKey:   "GO_ENV",
			envVal:   "production",
			expected: true,
		},
		{
			name:     "NODE_ENV set to production",
			envKey:   "NODE_ENV",
			envVal:   "production",
			expected: true,
		},
		{
			name:     "VERCEL_ENV set to production",
			envKey:   "VERCEL_ENV",
			envVal:   "production",
			expected: true,
		},
		{
			name:     "case insensitive",
			envKey:   "TG_ENV",
			envVal:   "ProDuCtioN",
			expected: true,
		},
		{
			name:     "whitespace trim",
			envKey:   "TG_ENV",
			envVal:   "  production  ",
			expected: true,
		},
		{
			name:     "non-production value",
			envKey:   "TG_ENV",
			envVal:   "development",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all checked env vars using t.Setenv to avoid global state mutation
			keys := []string{"TG_ENV", "APP_ENV", "GO_ENV", "NODE_ENV", "VERCEL_ENV"}
			for _, k := range keys {
				t.Setenv(k, "")
			}

			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			result := IsProduction()
			if result != tt.expected {
				t.Errorf("IsProduction() = %v, want %v", result, tt.expected)
			}
		})
	}
}
