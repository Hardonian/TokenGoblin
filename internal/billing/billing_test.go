package billing

import (
	"context"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// MockRepository implements storage.Repository for testing
type MockRepository struct {
	tenants map[string]*domain.Tenant
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		tenants: make(map[string]*domain.Tenant),
	}
}

func (m *MockRepository) UpsertTenant(ctx context.Context, tenant domain.Tenant) error {
	m.tenants[tenant.TenantID] = &tenant
	return nil
}

func (m *MockRepository) GetTenant(ctx context.Context, tenantID string) (*domain.Tenant, error) {
	if t, ok := m.tenants[tenantID]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *MockRepository) GetTenantByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Tenant, error) {
	for _, t := range m.tenants {
		if t.StripeCustomerID == stripeCustomerID {
			return t, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetTenantByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (*domain.Tenant, error) {
	for _, t := range m.tenants {
		if t.StripeSubscriptionID == stripeSubscriptionID {
			return t, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetTenantCurrentMonthCost(ctx context.Context, tenantID string) (float64, error) {
	return 0, nil
}

func (m *MockRepository) GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error) {
	return nil, nil
}

func (m *MockRepository) SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error {
	return nil
}

func (m *MockRepository) ListPricingOverrides(ctx context.Context, tenantID string) ([]domain.PricePoint, error) {
	return nil, nil
}

func (m *MockRepository) DeleteTenantData(ctx context.Context, tenantID string) error {
	delete(m.tenants, tenantID)
	return nil
}

func (m *MockRepository) DeleteOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	return 0, nil
}

func (m *MockRepository) SaveAPIKey(ctx context.Context, key domain.APIKey) error {
	return nil
}

func (m *MockRepository) GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error) {
	return nil, nil
}

func (m *MockRepository) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
	return nil
}

func (m *MockRepository) ListAPIKeys(ctx context.Context, tenantID string) ([]domain.APIKey, error) {
	return nil, nil
}

func (m *MockRepository) RevokeAPIKey(ctx context.Context, tenantID, keyID string) error {
	return nil
}

func (m *MockRepository) UpsertTenantMember(ctx context.Context, member domain.TenantMember) error {
	return nil
}

func (m *MockRepository) ListTenantMembers(ctx context.Context, tenantID string) ([]domain.TenantMember, error) {
	return nil, nil
}

func (m *MockRepository) SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	return nil
}

func (m *MockRepository) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error) {
	return nil, nil
}

func (m *MockRepository) SetRecommendationState(ctx context.Context, state domain.RecommendationState) error {
	return nil
}

func (m *MockRepository) ListRecommendationStates(ctx context.Context, tenantID string) ([]domain.RecommendationState, error) {
	return nil, nil
}

func (m *MockRepository) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	return nil
}

func (m *MockRepository) SaveCostSnapshot(ctx context.Context, snapshot domain.CostSnapshot) error {
	return nil
}

func (m *MockRepository) SaveAnomalySignal(ctx context.Context, signal domain.AnomalySignal) error {
	return nil
}

func (m *MockRepository) SaveOutputAnalysis(ctx context.Context, analysis domain.OutputAnalysis) error {
	return nil
}

func (m *MockRepository) SaveProductivitySummary(ctx context.Context, summary domain.ProductivitySummary) error {
	return nil
}

func (m *MockRepository) ListOutputAnalyses(ctx context.Context, tenantID string, limit int) ([]domain.OutputAnalysis, error) {
	return nil, nil
}

func (m *MockRepository) ListOutputAnalysesByWorker(ctx context.Context, tenantID, workerID string, limit int) ([]domain.OutputAnalysis, error) {
	return nil, nil
}

func (m *MockRepository) ListTokenEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error) {
	return nil, nil
}

func (m *MockRepository) ListTokenEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error) {
	return nil, nil
}

func (m *MockRepository) ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error) {
	return nil, nil
}

func (m *MockRepository) UpsertAgent(ctx context.Context, agent domain.Agent) error {
	return nil
}

func (m *MockRepository) ListAgents(ctx context.Context, tenantID string) ([]domain.Agent, error) {
	return nil, nil
}

func (m *MockRepository) UpsertGovernancePolicy(ctx context.Context, policy domain.GovernancePolicy) error {
	return nil
}

func (m *MockRepository) ListGovernancePolicies(ctx context.Context, tenantID string) ([]domain.GovernancePolicy, error) {
	return nil, nil
}

func (m *MockRepository) UpsertBudget(ctx context.Context, budget domain.Budget) error {
	return nil
}

func (m *MockRepository) ListBudgets(ctx context.Context, tenantID string) ([]domain.Budget, error) {
	return nil, nil
}

func (m *MockRepository) Close() error {
	return nil
}

func (m *MockRepository) Ping(ctx context.Context) error {
	return nil
}

var _ storage.Repository = (*MockRepository)(nil)

func TestPlanLimitUSD(t *testing.T) {
	t.Setenv("TG_PLAN_FREE_LIMIT_USD", "15.0")
	t.Setenv("TG_PLAN_PREMIUM_LIMIT_USD", "150.0")

	freeLimit := planLimitUSD(TierFree)
	premiumLimit := planLimitUSD(TierPremium)

	if freeLimit != 15.0 {
		t.Errorf("Expected free limit 15.0, got %f", freeLimit)
	}
	if premiumLimit != 150.0 {
		t.Errorf("Expected premium limit 150.0, got %f", premiumLimit)
	}
}

func TestPlanLimitUSDDefaults(t *testing.T) {
	t.Setenv("TG_PLAN_FREE_LIMIT_USD", "")
	t.Setenv("TG_PLAN_PREMIUM_LIMIT_USD", "")

	freeLimit := planLimitUSD(TierFree)
	premiumLimit := planLimitUSD(TierPremium)

	if freeLimit != 10.0 {
		t.Errorf("Expected default free limit 10.0, got %f", freeLimit)
	}
	if premiumLimit != 100.0 {
		t.Errorf("Expected default premium limit 100.0, got %f", premiumLimit)
	}
}

func TestTierForSubscriptionStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"active", "active", TierPremium},
		{"trialing", "trialing", TierPremium},
		{"past_due", "past_due", TierFree},
		{"canceled", "canceled", TierFree},
		{"unpaid", "unpaid", TierFree},
		{"incomplete", "incomplete", TierFree},
		{"incomplete_expired", "incomplete_expired", TierFree},
		{"unknown", "unknown", TierFree},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, _ := tierForSubscriptionStatus(tt.status)
			if tier != tt.expected {
				t.Errorf("tierForSubscriptionStatus(%q) = %q, want %q", tt.status, tier, tt.expected)
			}
		})
	}
}

func TestNormalizeVerifiedEvent(t *testing.T) {
	event := VerifiedStripeEvent{
		EventID:            "  evt_123  ",
		EventType:          "  customer.subscription.created  ",
		CustomerID:         "  cus_123  ",
		SubscriptionID:     "  sub_123  ",
		SubscriptionStatus: "  ACTIVE  ",
		TenantID:           "  tenant_123  ",
		Metadata: map[string]string{
			"tenant_id": "  meta_tenant_456  ",
		},
	}

	normalized := normalizeVerifiedEvent(event)

	if normalized.EventID != "evt_123" {
		t.Errorf("EventID not trimmed: %q", normalized.EventID)
	}
	if normalized.EventType != "customer.subscription.created" {
		t.Errorf("EventType not trimmed: %q", normalized.EventType)
	}
	if normalized.CustomerID != "cus_123" {
		t.Errorf("CustomerID not trimmed: %q", normalized.CustomerID)
	}
	if normalized.SubscriptionID != "sub_123" {
		t.Errorf("SubscriptionID not trimmed: %q", normalized.SubscriptionID)
	}
	if normalized.SubscriptionStatus != "active" {
		t.Errorf("SubscriptionStatus not lowercased: %q", normalized.SubscriptionStatus)
	}
	if normalized.TenantID != "tenant_123" {
		t.Errorf("TenantID not trimmed: %q", normalized.TenantID)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"first non-empty", []string{"", "second", "third"}, "second"},
		{"all empty", []string{"", "", ""}, ""},
		{"first is set", []string{"first", "second"}, "first"},
		{"whitespace trimmed", []string{"  ", "  value  "}, "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstNonEmpty(tt.input...)
			if result != tt.expected {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateStripeOwnership(t *testing.T) {
	tenant := domain.Tenant{
		TenantID:             "tenant_123",
		StripeCustomerID:     "cus_123",
		StripeSubscriptionID: "sub_123",
	}

	tests := []struct {
		name        string
		event       VerifiedStripeEvent
		expectError bool
	}{
		{"matching customer", VerifiedStripeEvent{CustomerID: "cus_123"}, false},
		{"mismatched customer", VerifiedStripeEvent{CustomerID: "cus_456"}, true},
		{"matching subscription", VerifiedStripeEvent{SubscriptionID: "sub_123"}, false},
		{"mismatched subscription", VerifiedStripeEvent{SubscriptionID: "sub_456"}, true},
		{"no customer in event", VerifiedStripeEvent{}, false},
		{"no subscription in event", VerifiedStripeEvent{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStripeOwnership(tenant, tt.event)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
