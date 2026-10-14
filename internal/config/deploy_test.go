package config

import (
	"strings"
	"testing"
)

func TestAllowDemoTenantAuth(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		expected bool
	}{
		{
			name:     "unset / empty",
			envVal:   "",
			expected: false,
		},
		{
			name:     "value 1",
			envVal:   "1",
			expected: true,
		},
		{
			name:     "value true",
			envVal:   "true",
			expected: true,
		},
		{
			name:     "value TRUE case-insensitive",
			envVal:   "TRUE",
			expected: true,
		},
		{
			name:     "value true with whitespace",
			envVal:   "  true  ",
			expected: true,
		},
		{
			name:     "value 0",
			envVal:   "0",
			expected: false,
		},
		{
			name:     "value false",
			envVal:   "false",
			expected: false,
		},
		{
			name:     "arbitrary value",
			envVal:   "enabled",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TG_ALLOW_DEMO_AUTH", tt.envVal)
			if got := AllowDemoTenantAuth(); got != tt.expected {
				t.Errorf("AllowDemoTenantAuth() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateServerEnv(t *testing.T) {
	allEnvKeys := []string{
		"TG_ENV",
		"APP_ENV",
		"GO_ENV",
		"NODE_ENV",
		"VERCEL_ENV",
		"TG_DB_DSN",
		"TG_INTERNAL_WEBHOOK_SECRET",
		"STRIPE_SECRET_KEY",
		"STRIPE_WEBHOOK_SECRET",
		"STRIPE_PRICE_PRO",
		"STRIPE_PRICE_ENTERPRISE",
		"TG_ALLOW_DEMO_AUTH",
	}

	validProdEnvs := map[string]string{
		"TG_ENV":                     "production",
		"TG_DB_DSN":                  "postgres://user:pass@localhost:5432/db",
		"TG_INTERNAL_WEBHOOK_SECRET": "secret_wh_123",
		"STRIPE_SECRET_KEY":          "sk_test_123",
		"STRIPE_WEBHOOK_SECRET":      "whsec_123",
		"STRIPE_PRICE_PRO":           "price_pro_123",
		"STRIPE_PRICE_ENTERPRISE":    "price_ent_123",
		"TG_ALLOW_DEMO_AUTH":         "false",
	}

	clearEnvs := func(t *testing.T) {
		for _, k := range allEnvKeys {
			t.Setenv(k, "")
		}
	}

	tests := []struct {
		name        string
		setupEnvs   map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:      "non-production env returns nil even with missing vars",
			setupEnvs: map[string]string{"TG_ENV": "development"},
			wantErr:   false,
		},
		{
			name:      "non-production env returns nil when empty",
			setupEnvs: map[string]string{},
			wantErr:   false,
		},
		{
			name:      "valid production environment",
			setupEnvs: validProdEnvs,
			wantErr:   false,
		},
		{
			name: "production with TG_ALLOW_DEMO_AUTH set to true fails unsafe check",
			setupEnvs: func() map[string]string {
				m := make(map[string]string)
				for k, v := range validProdEnvs {
					m[k] = v
				}
				m["TG_ALLOW_DEMO_AUTH"] = "true"
				return m
			}(),
			wantErr:     true,
			errContains: "production config unsafe: TG_ALLOW_DEMO_AUTH must be disabled",
		},
		{
			name: "production with TG_ALLOW_DEMO_AUTH set to 1 fails unsafe check",
			setupEnvs: func() map[string]string {
				m := make(map[string]string)
				for k, v := range validProdEnvs {
					m[k] = v
				}
				m["TG_ALLOW_DEMO_AUTH"] = "1"
				return m
			}(),
			wantErr:     true,
			errContains: "production config unsafe: TG_ALLOW_DEMO_AUTH must be disabled",
		},
		{
			name: "production missing TG_DB_DSN",
			setupEnvs: func() map[string]string {
				m := make(map[string]string)
				for k, v := range validProdEnvs {
					m[k] = v
				}
				delete(m, "TG_DB_DSN")
				return m
			}(),
			wantErr:     true,
			errContains: "TG_DB_DSN",
		},
		{
			name: "production missing TG_INTERNAL_WEBHOOK_SECRET",
			setupEnvs: func() map[string]string {
				m := make(map[string]string)
				for k, v := range validProdEnvs {
					m[k] = v
				}
				delete(m, "TG_INTERNAL_WEBHOOK_SECRET")
				return m
			}(),
			wantErr:     true,
			errContains: "TG_INTERNAL_WEBHOOK_SECRET",
		},
		{
			name: "production missing stripe billing env vars",
			setupEnvs: func() map[string]string {
				m := make(map[string]string)
				for k, v := range validProdEnvs {
					m[k] = v
				}
				delete(m, "STRIPE_SECRET_KEY")
				delete(m, "STRIPE_WEBHOOK_SECRET")
				delete(m, "STRIPE_PRICE_PRO")
				delete(m, "STRIPE_PRICE_ENTERPRISE")
				return m
			}(),
			wantErr:     true,
			errContains: "STRIPE_SECRET_KEY, STRIPE_WEBHOOK_SECRET, STRIPE_PRICE_PRO, STRIPE_PRICE_ENTERPRISE",
		},
		{
			name: "production missing all required env vars",
			setupEnvs: map[string]string{
				"TG_ENV": "production",
			},
			wantErr:     true,
			errContains: "TG_DB_DSN, TG_INTERNAL_WEBHOOK_SECRET, STRIPE_SECRET_KEY, STRIPE_WEBHOOK_SECRET, STRIPE_PRICE_PRO, STRIPE_PRICE_ENTERPRISE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvs(t)
			for k, v := range tt.setupEnvs {
				t.Setenv(k, v)
			}

			err := ValidateServerEnv()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateServerEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateServerEnv() error = %v, expected error containing %q", err, tt.errContains)
				}
			}
		})
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
