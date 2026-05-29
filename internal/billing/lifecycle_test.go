package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func TestProcessVerifiedStripeEventUpdatesTenantPlan(t *testing.T) {
	ctx := context.Background()
	repo, err := storage.OpenSQLite(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()

	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	if err := repo.UpsertTenant(ctx, domain.Tenant{
		TenantID:         "tenant-a",
		Name:             "Tenant A",
		Tier:             TierFree,
		UsageLimitUSD:    10,
		StripeCustomerID: "cus_123",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	result, err := ProcessVerifiedStripeEvent(ctx, repo, VerifiedStripeEvent{
		EventID:            "evt_1",
		EventType:          StripeEventSubscriptionUpdated,
		CustomerID:         "cus_123",
		SubscriptionID:     "sub_123",
		SubscriptionStatus: "active",
	}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("process event: %v", err)
	}
	if !result.Applied || result.Tier != TierPremium || result.UsageLimitUSD != 100 {
		t.Fatalf("expected premium applied result, got %+v", result)
	}

	tenant, err := repo.GetTenant(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if tenant.Tier != TierPremium || tenant.StripeSubscriptionID != "sub_123" {
		t.Fatalf("tenant billing not updated: %+v", tenant)
	}
}

func TestProcessVerifiedStripeEventDeletesSubscription(t *testing.T) {
	ctx := context.Background()
	repo, err := storage.OpenSQLite(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()

	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	if err := repo.UpsertTenant(ctx, domain.Tenant{
		TenantID:             "tenant-a",
		Name:                 "Tenant A",
		Tier:                 TierPremium,
		UsageLimitUSD:        100,
		StripeCustomerID:     "cus_123",
		StripeSubscriptionID: "sub_123",
		CreatedAt:            now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	result, err := ProcessVerifiedStripeEvent(ctx, repo, VerifiedStripeEvent{
		EventID:        "evt_2",
		EventType:      StripeEventSubscriptionDeleted,
		CustomerID:     "cus_123",
		SubscriptionID: "sub_123",
	}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("process event: %v", err)
	}
	if !result.Applied || result.Tier != TierFree || result.StripeSubscriptionID != "" {
		t.Fatalf("expected free deleted result, got %+v", result)
	}
}

func TestProcessVerifiedStripeEventRejectsCustomerMismatch(t *testing.T) {
	ctx := context.Background()
	repo, err := storage.OpenSQLite(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()

	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	if err := repo.UpsertTenant(ctx, domain.Tenant{
		TenantID:         "tenant-a",
		Name:             "Tenant A",
		Tier:             TierFree,
		UsageLimitUSD:    10,
		StripeCustomerID: "cus_real",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	_, err = ProcessVerifiedStripeEvent(ctx, repo, VerifiedStripeEvent{
		EventID:            "evt_3",
		EventType:          StripeEventSubscriptionUpdated,
		TenantID:           "tenant-a",
		CustomerID:         "cus_other",
		SubscriptionID:     "sub_123",
		SubscriptionStatus: "active",
	}, now.Add(time.Hour))
	if err == nil {
		t.Fatal("expected customer mismatch error")
	}
	var conflict StripeConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected StripeConflictError, got %T %[1]v", err)
	}
}
