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
