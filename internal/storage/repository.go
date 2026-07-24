package storage

import (
	"context"
	"errors"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

var ErrUnavailable = errors.New("database unavailable")

// Repository defines the persistent schema surface used by the execution layer.
type Repository interface {
	UpsertTenant(ctx context.Context, tenant domain.Tenant) error
	UpdateTenantWebhook(ctx context.Context, tenantID, webhookURL string) error
	GetTenant(ctx context.Context, tenantID string) (*domain.Tenant, error)
	GetTenantByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Tenant, error)
	GetTenantByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (*domain.Tenant, error)
	GetTenantCurrentMonthCost(ctx context.Context, tenantID string) (float64, error)
	GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error)
	SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error
	ListPricingOverrides(ctx context.Context, tenantID string) ([]domain.PricePoint, error)
	DeleteTenantData(ctx context.Context, tenantID string) error
	DeleteOldEvents(ctx context.Context, retentionDays int) (int64, error)
	SaveAPIKey(ctx context.Context, key domain.APIKey) error
	GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error
	ListAPIKeys(ctx context.Context, tenantID string) ([]domain.APIKey, error)
	RevokeAPIKey(ctx context.Context, keyID string, tenantID string) error
	UpsertTenantMember(ctx context.Context, member domain.TenantMember) error
	ListTenantMembers(ctx context.Context, tenantID string) ([]domain.TenantMember, error)
	SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error
	ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error)
	SetRecommendationState(ctx context.Context, state domain.RecommendationState) error
	ListRecommendationStates(ctx context.Context, tenantID string) ([]domain.RecommendationState, error)
	SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error
	SaveCostSnapshot(ctx context.Context, snapshot domain.CostSnapshot) error
	SaveAnomalySignal(ctx context.Context, signal domain.AnomalySignal) error
	SaveAnomalySignals(ctx context.Context, signals []domain.AnomalySignal) error
	SaveOutputAnalysis(ctx context.Context, analysis domain.OutputAnalysis) error
	SaveProductivitySummary(ctx context.Context, summary domain.ProductivitySummary) error
	ListOutputAnalyses(ctx context.Context, tenantID string, limit int) ([]domain.OutputAnalysis, error)
	ListOutputAnalysesByWorker(ctx context.Context, tenantID, workerID string, limit int) ([]domain.OutputAnalysis, error)
	ListTokenEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error)
	ListTokenEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error)
	ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error)

	// Founder Mode extensions
	UpsertAgent(ctx context.Context, agent domain.Agent) error
	ListAgents(ctx context.Context, tenantID string) ([]domain.Agent, error)
	UpsertGovernancePolicy(ctx context.Context, policy domain.GovernancePolicy) error
	ListGovernancePolicies(ctx context.Context, tenantID string) ([]domain.GovernancePolicy, error)
	UpsertBudget(ctx context.Context, budget domain.Budget) error
	ListBudgets(ctx context.Context, tenantID string) ([]domain.Budget, error)

	GetTuningProfile(ctx context.Context, tenantID string) (*domain.TuningProfile, error)
	UpsertTuningProfile(ctx context.Context, profile domain.TuningProfile) error
	GetUnexportedEvents(ctx context.Context, limit int) ([]domain.TokenEvent, error)
	MarkEventsExported(ctx context.Context, eventIDs []string) error

	// Health check
	Ping(ctx context.Context) error

	Close() error
}

type UnavailableRepository struct {
	Cause error
}

func NewUnavailableRepository(cause error) *UnavailableRepository {
	return &UnavailableRepository{Cause: cause}
}

func (r *UnavailableRepository) err() error {
	if r.Cause == nil {
		return ErrUnavailable
	}
	return errors.Join(ErrUnavailable, r.Cause)
}

func (r *UnavailableRepository) UpsertTenant(context.Context, domain.Tenant) error {
	return r.err()
}

func (r *UnavailableRepository) UpdateTenantWebhook(context.Context, string, string) error {
	return r.err()
}

func (r *UnavailableRepository) GetTenant(context.Context, string) (*domain.Tenant, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) GetTenantByStripeCustomerID(context.Context, string) (*domain.Tenant, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) GetTenantByStripeSubscriptionID(context.Context, string) (*domain.Tenant, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) GetTenantCurrentMonthCost(context.Context, string) (float64, error) {
	return 0, r.err()
}

func (r *UnavailableRepository) GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error {
	return r.err()
}

func (r *UnavailableRepository) ListPricingOverrides(ctx context.Context, tenantID string) ([]domain.PricePoint, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) DeleteTenantData(context.Context, string) error {
	return r.err()
}

func (r *UnavailableRepository) DeleteOldEvents(context.Context, int) (int64, error) {
	return 0, r.err()
}

func (r *UnavailableRepository) SaveAPIKey(context.Context, domain.APIKey) error {
	return r.err()
}

func (r *UnavailableRepository) GetAPIKey(context.Context, string) (*domain.APIKey, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) UpdateAPIKeyLastUsed(context.Context, string) error {
	return r.err()
}

func (r *UnavailableRepository) ListAPIKeys(context.Context, string) ([]domain.APIKey, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) RevokeAPIKey(context.Context, string, string) error {
	return r.err()
}

func (r *UnavailableRepository) UpsertTenantMember(context.Context, domain.TenantMember) error {
	return r.err()
}

func (r *UnavailableRepository) ListTenantMembers(context.Context, string) ([]domain.TenantMember, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) SaveAuditEvent(context.Context, domain.AuditEvent) error {
	return r.err()
}

func (r *UnavailableRepository) ListAuditEvents(context.Context, string, int) ([]domain.AuditEvent, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) SetRecommendationState(context.Context, domain.RecommendationState) error {
	return r.err()
}

func (r *UnavailableRepository) ListRecommendationStates(context.Context, string) ([]domain.RecommendationState, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) SaveTokenEvent(context.Context, domain.TokenEvent) error {
	return r.err()
}

func (r *UnavailableRepository) SaveCostSnapshot(context.Context, domain.CostSnapshot) error {
	return r.err()
}

func (r *UnavailableRepository) SaveAnomalySignal(context.Context, domain.AnomalySignal) error {
	return r.err()
}

func (r *UnavailableRepository) SaveAnomalySignals(context.Context, []domain.AnomalySignal) error {
	return r.err()
}

func (r *UnavailableRepository) SaveOutputAnalysis(context.Context, domain.OutputAnalysis) error {
	return r.err()
}

func (r *UnavailableRepository) SaveProductivitySummary(context.Context, domain.ProductivitySummary) error {
	return r.err()
}

func (r *UnavailableRepository) ListOutputAnalyses(context.Context, string, int) ([]domain.OutputAnalysis, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) ListOutputAnalysesByWorker(context.Context, string, string, int) ([]domain.OutputAnalysis, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) ListTokenEvents(context.Context, string, int) ([]domain.TokenEvent, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) ListTokenEventsBefore(context.Context, string, time.Time, int) ([]domain.TokenEvent, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error) {
	return nil, r.Cause
}

func (r *UnavailableRepository) UpsertAgent(ctx context.Context, agent domain.Agent) error {
	return r.Cause
}

func (r *UnavailableRepository) ListAgents(ctx context.Context, tenantID string) ([]domain.Agent, error) {
	return nil, r.Cause
}

func (r *UnavailableRepository) UpsertGovernancePolicy(ctx context.Context, policy domain.GovernancePolicy) error {
	return r.Cause
}

func (r *UnavailableRepository) ListGovernancePolicies(ctx context.Context, tenantID string) ([]domain.GovernancePolicy, error) {
	return nil, r.Cause
}

func (r *UnavailableRepository) UpsertBudget(ctx context.Context, budget domain.Budget) error {
	return r.Cause
}

func (r *UnavailableRepository) ListBudgets(ctx context.Context, tenantID string) ([]domain.Budget, error) {
	return nil, r.Cause
}

func (r *UnavailableRepository) Close() error {
	return r.Cause
}

func (r *UnavailableRepository) GetTuningProfile(ctx context.Context, tenantID string) (*domain.TuningProfile, error) {
	return nil, r.Cause
}

func (r *UnavailableRepository) UpsertTuningProfile(ctx context.Context, p domain.TuningProfile) error {
	return r.Cause
}

func (r *UnavailableRepository) GetUnexportedEvents(ctx context.Context, limit int) ([]domain.TokenEvent, error) {
	return nil, r.Cause
}

func (r *UnavailableRepository) MarkEventsExported(ctx context.Context, eventIDs []string) error {
	return r.Cause
}

func (r *UnavailableRepository) Ping(ctx context.Context) error {
	return r.Cause
}
