package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/billing"
	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalBillingStripeEvent_Unauthorized(t *testing.T) {
	t.Setenv("TG_INTERNAL_WEBHOOK_SECRET", "super-secret-token")

	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "auth_test.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	router := NewRouter(service, repo, nil)

	body := []byte(`{"event_id":"evt_1","event_type":"customer.subscription.updated"}`)

	// Case 1: Missing Authorization header
	req := httptest.NewRequest(http.MethodPost, "/internal/billing/stripe-event", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Case 2: Invalid Bearer token
	req = httptest.NewRequest(http.MethodPost, "/internal/billing/stripe-event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestInternalBillingStripeEvent_Idempotency(t *testing.T) {
	t.Setenv("TG_INTERNAL_WEBHOOK_SECRET", "test-secret")

	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "idem_test.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	now := time.Now().UTC()
	err = repo.UpsertTenant(context.Background(), domain.Tenant{
		TenantID:         "tenant-idem",
		Name:             "Idempotent Tenant",
		Tier:             billing.TierFree,
		UsageLimitUSD:    10.0,
		StripeCustomerID: "cus_idem_123",
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	require.NoError(t, err)

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	router := NewRouter(service, repo, nil)

	payload := `{
		"event_id": "evt_idem_unique_101",
		"event_type": "customer.subscription.updated",
		"customer_id": "cus_idem_123",
		"subscription_id": "sub_idem_abc",
		"subscription_status": "active"
	}`

	// First attempt: should apply
	req1 := httptest.NewRequest(http.MethodPost, "/internal/billing/stripe-event", strings.NewReader(payload))
	req1.Header.Set("Authorization", "Bearer test-secret")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	require.Equal(t, http.StatusOK, rec1.Code)
	var resp1 Envelope
	require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &resp1))
	assert.Equal(t, "success", resp1.Status)

	// Second attempt with identical event_id: should be ignored (idempotent)
	req2 := httptest.NewRequest(http.MethodPost, "/internal/billing/stripe-event", strings.NewReader(payload))
	req2.Header.Set("Authorization", "Bearer test-secret")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code)
	var resp2 Envelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp2))
	assert.Equal(t, "ignored", resp2.Status)
}

func TestInternalBillingStripeEvent_InvoicePaymentFailed_Downgrades(t *testing.T) {
	t.Setenv("TG_INTERNAL_WEBHOOK_SECRET", "test-secret")

	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "downgrade_test.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	now := time.Now().UTC()
	err = repo.UpsertTenant(context.Background(), domain.Tenant{
		TenantID:             "tenant-paid",
		Name:                 "Paid Tenant",
		Tier:                 billing.TierPro,
		UsageLimitUSD:        100.0,
		StripeCustomerID:     "cus_paid_999",
		StripeSubscriptionID: "sub_paid_999",
		CreatedAt:            now,
		UpdatedAt:            now,
	})
	require.NoError(t, err)

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	router := NewRouter(service, repo, nil)

	payload := `{
		"event_id": "evt_inv_failed_101",
		"event_type": "invoice.payment_failed",
		"customer_id": "cus_paid_999",
		"subscription_id": "sub_paid_999",
		"subscription_status": "past_due"
	}`

	req := httptest.NewRequest(http.MethodPost, "/internal/billing/stripe-event", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify tenant was downgraded to free tier and usage limit revoked to free limit
	updated, err := repo.GetTenant(context.Background(), "tenant-paid")
	require.NoError(t, err)
	assert.Equal(t, billing.TierFree, updated.Tier)
	assert.Equal(t, 10.0, updated.UsageLimitUSD)
}

func TestStripeWebhook_RawBodySignatureVerification(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_secret_abc")

	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "sig_test.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	router := NewRouter(service, repo, nil)

	payload := `{"id":"evt_sig_1","type":"customer.subscription.updated"}`

	// Case 1: Missing Stripe-Signature header
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", strings.NewReader(payload))
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	assert.Equal(t, http.StatusBadRequest, rec1.Code)
	var resp1 Envelope
	require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &resp1))
	assert.Equal(t, "invalid_signature", resp1.Error.Code)

	// Case 2: Invalid signature header
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", strings.NewReader(payload))
	req2.Header.Set("Stripe-Signature", "t=123456,v1=bad_signature_hash")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusBadRequest, rec2.Code)
	var resp2 Envelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp2))
	assert.Equal(t, "invalid_signature", resp2.Error.Code)
}

func TestStripeWebhook_MaxBodyBytesExceeded(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_secret_abc")

	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "maxbytes_test.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	router := NewRouter(service, repo, nil)

	// Payload larger than 64KB (65536 bytes)
	largeBody := strings.Repeat("A", 70000)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", strings.NewReader(largeBody))
	req.Header.Set("Stripe-Signature", "t=123,v1=abc")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "read_error", resp.Error.Code)
}
