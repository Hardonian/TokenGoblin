package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// BillingStatus describes the current billing state of a tenant.
type BillingStatus struct {
	TenantID            string  `json:"tenant_id"`
	Tier                string  `json:"tier"`
	StripeCustomerID    string  `json:"stripe_customer_id,omitempty"`
	CurrentMonthCostUSD float64 `json:"current_month_cost_usd"`
	UsageLimitUSD       float64 `json:"usage_limit_usd"`
	UsagePercent        float64 `json:"usage_percent"`
	SubscriptionID      string  `json:"subscription_id,omitempty"`
	NeedsUpgrade        bool    `json:"needs_upgrade"`
	NearLimit           bool    `json:"near_limit"`
	AtLimit             bool    `json:"at_limit"`
}

func stripeSecretKey() string {
	return strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
}

func stripeBaseURL() string {
	return "https://api.stripe.com/v1"
}

func stripePost(path string, data url.Values) ([]byte, error) {
	secret := stripeSecretKey()
	if secret == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is not set")
	}

	fullURL := stripeBaseURL() + path
	body := data.Encode()

	req, err := http.NewRequest("POST", fullURL, bytes.NewBufferString(body))
	if err != nil {
		return nil, fmt.Errorf("create stripe request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read stripe response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func stripeGet(path string) ([]byte, error) {
	secret := stripeSecretKey()
	if secret == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is not set")
	}

	fullURL := stripeBaseURL() + path

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create stripe request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read stripe response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session for the tenant and
// returns the hosted checkout URL and session ID.
func CreateCheckoutSession(ctx context.Context, repo storage.Repository, tenantID, successURL, cancelURL, priceID string) (string, string, error) {
	tenant, err := repo.GetTenant(ctx, tenantID)
	if err != nil {
		return "", "", fmt.Errorf("get tenant: %w", err)
	}
	if tenant == nil {
		return "", "", fmt.Errorf("tenant not found: %s", tenantID)
	}

	customerID := tenant.StripeCustomerID

	// If the tenant doesn't have a Stripe customer ID yet, create one.
	if customerID == "" {
		custResp, err := stripePost("/customers", url.Values{
			"metadata[tenant_id]": {tenantID},
			"name":                {tenant.Name},
		})
		if err != nil {
			return "", "", fmt.Errorf("create stripe customer: %w", err)
		}
		var custResult struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(custResp, &custResult); err != nil {
			return "", "", fmt.Errorf("parse stripe customer response: %w", err)
		}
		customerID = custResult.ID

		// Persist the customer ID on the tenant
		tenant.StripeCustomerID = customerID
		tenant.UpdatedAt = time.Now().UTC()
		if err := repo.UpsertTenant(ctx, *tenant); err != nil {
			slog.Warn("failed to update tenant with stripe customer ID", "tenant_id", tenantID, "error", err)
		}
	}

	// Create a checkout session
	vals := url.Values{
		"mode":                                   {"subscription"},
		"customer":                               {customerID},
		"success_url":                            {successURL},
		"cancel_url":                             {cancelURL},
		"line_items[0][price]":                   {priceID},
		"line_items[0][quantity]":                {"1"},
		"metadata[tenant_id]":                    {tenantID},
		"client_reference_id":                    {tenantID},
		"subscription_data[metadata][tenant_id]": {tenantID},
	}

	sessionResp, err := stripePost("/checkout/sessions", vals)
	if err != nil {
		return "", "", fmt.Errorf("create checkout session: %w", err)
	}

	var sessionResult struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(sessionResp, &sessionResult); err != nil {
		return "", "", fmt.Errorf("parse checkout session response: %w", err)
	}

	if sessionResult.URL == "" {
		return "", "", fmt.Errorf("stripe returned empty checkout URL")
	}

	return sessionResult.URL, sessionResult.ID, nil
}

// CreatePortalSession creates a Stripe Billing Portal session for the tenant.
func CreatePortalSession(ctx context.Context, repo storage.Repository, tenantID, returnURL string) (string, error) {
	tenant, err := repo.GetTenant(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("get tenant: %w", err)
	}
	if tenant == nil {
		return "", fmt.Errorf("tenant not found: %s", tenantID)
	}
	if tenant.StripeCustomerID == "" {
		return "", fmt.Errorf("tenant has no Stripe customer ID; cannot create portal session")
	}

	vals := url.Values{
		"customer":   {tenant.StripeCustomerID},
		"return_url": {returnURL},
	}

	portalResp, err := stripePost("/billing_portal/sessions", vals)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}

	var portalResult struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(portalResp, &portalResult); err != nil {
		return "", fmt.Errorf("parse portal session response: %w", err)
	}

	if portalResult.URL == "" {
		return "", fmt.Errorf("stripe returned empty portal URL")
	}

	return portalResult.URL, nil
}

// GetBillingStatus computes the current billing status for a tenant.
func GetBillingStatus(ctx context.Context, repo storage.Repository, tenantID string) (*BillingStatus, error) {
	tenant, err := repo.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}

	currentMonthCost, err := repo.GetTenantCurrentMonthCost(ctx, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrUnavailable) {
			currentMonthCost = 0
		} else {
			return nil, fmt.Errorf("get current month cost: %w", err)
		}
	}

	usagePercent := 0.0
	if tenant.UsageLimitUSD > 0 {
		usagePercent = (currentMonthCost / tenant.UsageLimitUSD) * 100
	}
	if usagePercent > 100 {
		usagePercent = 100
	}

	// near_limit at 80%, at_limit at 100%
	nearLimit := usagePercent >= 80.0 && usagePercent < 100.0
	atLimit := usagePercent >= 100.0

	status := &BillingStatus{
		TenantID:            tenant.TenantID,
		Tier:                tenant.Tier,
		StripeCustomerID:    tenant.StripeCustomerID,
		CurrentMonthCostUSD: currentMonthCost,
		UsageLimitUSD:       tenant.UsageLimitUSD,
		UsagePercent:        usagePercent,
		SubscriptionID:      tenant.StripeSubscriptionID,
		NearLimit:           nearLimit,
		AtLimit:             atLimit,
		NeedsUpgrade:        tenant.Tier == TierFree && atLimit,
	}

	return status, nil
}

// RegisterTenant creates a new tenant and returns the tenant plus a generated API key.
func RegisterTenant(ctx context.Context, repo storage.Repository, tenantID, name string) (*domain.Tenant, string, error) {
	if tenantID == "" {
		return nil, "", fmt.Errorf("tenant_id is required")
	}
	if name == "" {
		return nil, "", fmt.Errorf("name is required")
	}

	existing, err := repo.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("check existing tenant: %w", err)
	}
	if existing != nil {
		return nil, "", fmt.Errorf("tenant already exists: %s", tenantID)
	}

	now := time.Now().UTC()
	tenant := &domain.Tenant{
		TenantID:      tenantID,
		Name:          name,
		Tier:          TierFree,
		UsageLimitUSD: planLimitUSD(TierFree),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := repo.UpsertTenant(ctx, *tenant); err != nil {
		return nil, "", fmt.Errorf("save tenant: %w", err)
	}

	// Generate API key
	apiKey, compoundToken, err := moat.GenerateAPIKey(tenantID, "default")
	if err != nil {
		return nil, "", fmt.Errorf("generate api key: %w", err)
	}

	if err := repo.SaveAPIKey(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("save api key: %w", err)
	}

	_ = repo.SaveAuditEvent(ctx, domain.AuditEvent{
		EventID:   "aud_" + randomHex(12),
		TenantID:  tenantID,
		Type:      "tenant.registered",
		Actor:     "system:registration",
		Resource:  "tenant:" + tenantID,
		Timestamp: now,
	})

	return tenant, compoundToken, nil
}
