package billing

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

const (
	StripeEventSubscriptionCreated = "customer.subscription.created"
	StripeEventSubscriptionUpdated = "customer.subscription.updated"
	StripeEventSubscriptionDeleted = "customer.subscription.deleted"
	StripeEventCheckoutCompleted   = "checkout.session.completed"

	TierFree    = "free"
	TierPremium = "premium"
)

type VerifiedStripeEvent struct {
	EventID            string            `json:"event_id"`
	EventType          string            `json:"event_type"`
	CustomerID         string            `json:"customer_id,omitempty"`
	SubscriptionID     string            `json:"subscription_id,omitempty"`
	SubscriptionStatus string            `json:"subscription_status,omitempty"`
	TenantID           string            `json:"tenant_id,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

type StripeLifecycleResult struct {
	EventID              string  `json:"event_id"`
	EventType            string  `json:"event_type"`
	Action               string  `json:"action"`
	Applied              bool    `json:"applied"`
	TenantID             string  `json:"tenant_id,omitempty"`
	Tier                 string  `json:"tier,omitempty"`
	UsageLimitUSD        float64 `json:"usage_limit_usd,omitempty"`
	StripeCustomerID     string  `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string  `json:"stripe_subscription_id,omitempty"`
}

type StripeValidationError struct {
	Message string
}

func (e StripeValidationError) Error() string {
	return e.Message
}

type StripeConflictError struct {
	Message string
}

func (e StripeConflictError) Error() string {
	return e.Message
}

type StripeSyncer struct {
	repo   storage.Repository
	logger *slog.Logger
}

func NewStripeSyncer(repo storage.Repository, logger *slog.Logger) *StripeSyncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &StripeSyncer{
		repo:   repo,
		logger: logger,
	}
}

func (s *StripeSyncer) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncAllTenants(ctx)
		}
	}
}

func (s *StripeSyncer) syncAllTenants(ctx context.Context) {
	s.logger.Info("Starting Stripe usage sync")
	// In a real implementation, we would query all tenants,
	// fetch their Stripe IDs, aggregate their usage since the last sync,
	// and send it to the Stripe API.

	// Example stub:
	// tenants, _ := s.repo.ListTenants(ctx)
	// for _, t := range tenants {
	//    usage, _ := s.repo.GetTenantCurrentMonthCost(ctx, t.TenantID)
	//    // POST https://api.stripe.com/v1/subscription_items/{id}/usage_records
	// }

	s.logger.Info("Stripe usage sync completed")
}

func ProcessVerifiedStripeEvent(ctx context.Context, repo storage.Repository, event VerifiedStripeEvent, now time.Time) (StripeLifecycleResult, error) {
	event = normalizeVerifiedEvent(event)
	result := StripeLifecycleResult{
		EventID:   event.EventID,
		EventType: event.EventType,
		Action:    "ignored_unhandled_event",
	}
	if event.EventID == "" {
		return result, StripeValidationError{Message: "event_id is required"}
	}
	if event.EventType == "" {
		return result, StripeValidationError{Message: "event_type is required"}
	}

	switch event.EventType {
	case StripeEventSubscriptionCreated, StripeEventSubscriptionUpdated:
		if event.CustomerID == "" || event.SubscriptionID == "" || event.SubscriptionStatus == "" {
			return result, StripeValidationError{Message: "subscription lifecycle events require customer_id, subscription_id, and subscription_status"}
		}
		return applySubscriptionLifecycle(ctx, repo, event, now)
	case StripeEventSubscriptionDeleted:
		if event.CustomerID == "" && event.SubscriptionID == "" {
			return result, StripeValidationError{Message: "subscription deletion requires customer_id or subscription_id"}
		}
		return applySubscriptionLifecycle(ctx, repo, event, now)
	case StripeEventCheckoutCompleted:
		return linkCheckoutSession(ctx, repo, event, now)
	default:
		return result, nil
	}
}

func applySubscriptionLifecycle(ctx context.Context, repo storage.Repository, event VerifiedStripeEvent, now time.Time) (StripeLifecycleResult, error) {
	result := StripeLifecycleResult{
		EventID:   event.EventID,
		EventType: event.EventType,
		Action:    "ignored_no_matching_tenant",
	}
	tenant, err := resolveTenant(ctx, repo, event)
	if err != nil {
		return result, err
	}
	if tenant == nil {
		return result, nil
	}
	if err := validateStripeOwnership(*tenant, event); err != nil {
		return result, err
	}

	tenant.StripeCustomerID = firstNonEmpty(tenant.StripeCustomerID, event.CustomerID)
	if event.EventType == StripeEventSubscriptionDeleted {
		tenant.StripeSubscriptionID = ""
		tenant.Tier = TierFree
		tenant.UsageLimitUSD = planLimitUSD(TierFree)
		result.Action = "subscription_deleted"
	} else {
		tenant.StripeSubscriptionID = event.SubscriptionID
		tenant.Tier, tenant.UsageLimitUSD = tierForSubscriptionStatus(event.SubscriptionStatus)
		result.Action = "subscription_" + strings.TrimPrefix(event.EventType, "customer.subscription.")
	}
	tenant.UpdatedAt = now.UTC()
	if err := repo.UpsertTenant(ctx, *tenant); err != nil {
		return result, err
	}
	_ = repo.SaveAuditEvent(ctx, stripeAuditEvent(*tenant, event, result.Action, now))

	result.Applied = true
	result.TenantID = tenant.TenantID
	result.Tier = tenant.Tier
	result.UsageLimitUSD = tenant.UsageLimitUSD
	result.StripeCustomerID = tenant.StripeCustomerID
	result.StripeSubscriptionID = tenant.StripeSubscriptionID
	return result, nil
}

func linkCheckoutSession(ctx context.Context, repo storage.Repository, event VerifiedStripeEvent, now time.Time) (StripeLifecycleResult, error) {
	result := StripeLifecycleResult{
		EventID:   event.EventID,
		EventType: event.EventType,
		Action:    "ignored_no_tenant_reference",
	}
	if event.TenantID == "" {
		return result, nil
	}
	tenant, err := repo.GetTenant(ctx, event.TenantID)
	if err != nil {
		return result, err
	}
	if tenant == nil {
		result.Action = "ignored_no_matching_tenant"
		return result, nil
	}
	if err := validateStripeOwnership(*tenant, event); err != nil {
		return result, err
	}

	tenant.StripeCustomerID = firstNonEmpty(tenant.StripeCustomerID, event.CustomerID)
	tenant.StripeSubscriptionID = firstNonEmpty(tenant.StripeSubscriptionID, event.SubscriptionID)
	tenant.UpdatedAt = now.UTC()
	if err := repo.UpsertTenant(ctx, *tenant); err != nil {
		return result, err
	}
	_ = repo.SaveAuditEvent(ctx, stripeAuditEvent(*tenant, event, "checkout_session_linked", now))

	result.Action = "checkout_session_linked"
	result.Applied = true
	result.TenantID = tenant.TenantID
	result.Tier = tenant.Tier
	result.UsageLimitUSD = tenant.UsageLimitUSD
	result.StripeCustomerID = tenant.StripeCustomerID
	result.StripeSubscriptionID = tenant.StripeSubscriptionID
	return result, nil
}

func resolveTenant(ctx context.Context, repo storage.Repository, event VerifiedStripeEvent) (*domain.Tenant, error) {
	if event.TenantID != "" {
		return repo.GetTenant(ctx, event.TenantID)
	}
	if event.CustomerID != "" {
		tenant, err := repo.GetTenantByStripeCustomerID(ctx, event.CustomerID)
		if err != nil || tenant != nil {
			return tenant, err
		}
	}
	if event.SubscriptionID != "" {
		return repo.GetTenantByStripeSubscriptionID(ctx, event.SubscriptionID)
	}
	return nil, nil
}

func validateStripeOwnership(tenant domain.Tenant, event VerifiedStripeEvent) error {
	if event.CustomerID != "" && tenant.StripeCustomerID != "" && tenant.StripeCustomerID != event.CustomerID {
		return StripeConflictError{Message: "stripe customer does not match tenant billing owner"}
	}
	if event.SubscriptionID != "" && tenant.StripeSubscriptionID != "" && tenant.StripeSubscriptionID != event.SubscriptionID {
		return StripeConflictError{Message: "stripe subscription does not match tenant billing owner"}
	}
	return nil
}

func normalizeVerifiedEvent(event VerifiedStripeEvent) VerifiedStripeEvent {
	event.EventID = strings.TrimSpace(event.EventID)
	event.EventType = strings.TrimSpace(event.EventType)
	event.CustomerID = strings.TrimSpace(event.CustomerID)
	event.SubscriptionID = strings.TrimSpace(event.SubscriptionID)
	event.SubscriptionStatus = strings.ToLower(strings.TrimSpace(event.SubscriptionStatus))
	event.TenantID = strings.TrimSpace(firstNonEmpty(event.TenantID, event.Metadata["tenant_id"], event.Metadata["tenantId"]))
	return event
}

func tierForSubscriptionStatus(status string) (string, float64) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "trialing":
		return TierPremium, planLimitUSD(TierPremium)
	default:
		return TierFree, planLimitUSD(TierFree)
	}
}

func planLimitUSD(tier string) float64 {
	envName := "TG_PLAN_FREE_LIMIT_USD"
	defaultValue := 10.0
	if tier == TierPremium {
		envName = "TG_PLAN_PREMIUM_LIMIT_USD"
		defaultValue = 100.0
	}
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || parsed < 0 {
		return defaultValue
	}
	return parsed
}

func stripeAuditEvent(tenant domain.Tenant, event VerifiedStripeEvent, action string, now time.Time) domain.AuditEvent {
	return domain.AuditEvent{
		EventID:  "aud_" + randomHex(12),
		TenantID: tenant.TenantID,
		Type:     "billing.stripe." + action,
		Actor:    "stripe:webhook",
		Resource: "stripe_event:" + event.EventID,
		Metadata: map[string]interface{}{
			"stripe_event_type":      event.EventType,
			"stripe_customer_id":     event.CustomerID,
			"stripe_subscription_id": event.SubscriptionID,
			"subscription_status":    event.SubscriptionStatus,
			"tier":                   tenant.Tier,
			"usage_limit_usd":        tenant.UsageLimitUSD,
		},
		Timestamp: now.UTC(),
	}
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buffer)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// VerifySignature validates a Stripe webhook signature.
func VerifySignature(payload []byte, sigHeader string, secret string, tolerance time.Duration) error {
	if secret == "" {
		return errors.New("missing stripe webhook secret")
	}
	if sigHeader == "" {
		return errors.New("missing Stripe-Signature header")
	}

	var timestampStr string
	var signatures []string

	pairs := strings.Split(sigHeader, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		switch key {
		case "t":
			timestampStr = val
		case "v1":
			signatures = append(signatures, val)
		}
	}

	if timestampStr == "" {
		return errors.New("missing timestamp in signature header")
	}
	if len(signatures) == 0 {
		return errors.New("missing v1 signature in signature header")
	}

	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	// Replay attack prevention
	timestamp := time.Unix(timestampInt, 0)
	if tolerance > 0 {
		diff := time.Since(timestamp)
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			return fmt.Errorf("signature timestamp %v is outside of tolerance %v", timestamp, tolerance)
		}
	}

	// Compute expected signature
	macPayload := []byte(timestampStr + "." + string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(macPayload)
	expectedMAC := mac.Sum(nil)

	// Compare with signatures in header
	for _, sig := range signatures {
		sigBytes, err := hex.DecodeString(sig)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare(expectedMAC, sigBytes) == 1 {
			return nil // Valid signature found
		}
	}

	return errors.New("signature is invalid")
}
