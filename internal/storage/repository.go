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
	GetTenant(ctx context.Context, tenantID string) (*domain.Tenant, error)
	GetTenantCurrentMonthCost(ctx context.Context, tenantID string) (float64, error)
	GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error)
	SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error
	DeleteTenantData(ctx context.Context, tenantID string) error
	DeleteOldEvents(ctx context.Context, retentionDays int) (int64, error)
	SaveAPIKey(ctx context.Context, key domain.APIKey) error
	GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error
	SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error
	SaveCostSnapshot(ctx context.Context, snapshot domain.CostSnapshot) error
	SaveAnomalySignal(ctx context.Context, signal domain.AnomalySignal) error
	SaveProductivitySummary(ctx context.Context, summary domain.ProductivitySummary) error
	ListTokenEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error)
	ListTokenEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error)
	ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error)
<<<<<<< HEAD
	ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error)
=======
>>>>>>> 7b9f8e1773f8de964d6ecbbf953205607f2070d4
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

func (r *UnavailableRepository) GetTenant(context.Context, string) (*domain.Tenant, error) {
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

func (r *UnavailableRepository) DeleteTenantData(ctx context.Context, tenantID string) error {
	return r.err()
}

func (r *UnavailableRepository) DeleteOldEvents(ctx context.Context, retentionDays int) (int64, error) {
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

func (r *UnavailableRepository) SaveTokenEvent(context.Context, domain.TokenEvent) error {
	return r.err()
}

func (r *UnavailableRepository) SaveCostSnapshot(context.Context, domain.CostSnapshot) error {
	return r.err()
}

func (r *UnavailableRepository) SaveAnomalySignal(context.Context, domain.AnomalySignal) error {
	return r.err()
}

func (r *UnavailableRepository) SaveProductivitySummary(context.Context, domain.ProductivitySummary) error {
	return r.err()
}

func (r *UnavailableRepository) ListTokenEvents(context.Context, string, int) ([]domain.TokenEvent, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) ListTokenEventsBefore(context.Context, string, time.Time, int) ([]domain.TokenEvent, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) ListAnomalySignals(context.Context, string, int) ([]domain.AnomalySignal, error) {
	return nil, r.err()
}

<<<<<<< HEAD
func (r *UnavailableRepository) ListAnomalySignals(context.Context, string, int) ([]domain.AnomalySignal, error) {
	return nil, r.err()
}
=======
>>>>>>> 7b9f8e1773f8de964d6ecbbf953205607f2070d4

func (r *UnavailableRepository) Close() error {
	return nil
}
