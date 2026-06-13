package config

import (
	"fmt"
	"os"
	"strings"
)

func IsProduction() bool {
	for _, key := range []string{"TG_ENV", "APP_ENV", "GO_ENV", "NODE_ENV", "VERCEL_ENV"} {
		if strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "production") {
			return true
		}
	}
	return false
}

func AllowDemoTenantAuth() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("TG_ALLOW_DEMO_AUTH")), "1") || strings.EqualFold(strings.TrimSpace(os.Getenv("TG_ALLOW_DEMO_AUTH")), "true")
}

func ValidateServerEnv() error {
	if !IsProduction() {
		return nil
	}
	var missing []string
	if strings.TrimSpace(os.Getenv("TG_DB_DSN")) == "" {
		missing = append(missing, "TG_DB_DSN")
	}
	if strings.TrimSpace(os.Getenv("TG_INTERNAL_WEBHOOK_SECRET")) == "" {
		missing = append(missing, "TG_INTERNAL_WEBHOOK_SECRET")
	}
	// Stripe billing configuration (required in production)
	if strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")) == "" {
		missing = append(missing, "STRIPE_SECRET_KEY")
	}
	if strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")) == "" {
		missing = append(missing, "STRIPE_WEBHOOK_SECRET")
	}
	if strings.TrimSpace(os.Getenv("STRIPE_PRICE_PRO")) == "" {
		missing = append(missing, "STRIPE_PRICE_PRO")
	}
	if strings.TrimSpace(os.Getenv("STRIPE_PRICE_ENTERPRISE")) == "" {
		missing = append(missing, "STRIPE_PRICE_ENTERPRISE")
	}
	if AllowDemoTenantAuth() {
		return fmt.Errorf("production config unsafe: TG_ALLOW_DEMO_AUTH must be disabled")
	}
	if len(missing) > 0 {
		return fmt.Errorf("production config missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}
