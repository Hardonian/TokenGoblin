package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func TestTokenUsageRouteRejectsCrossTenantWrites(t *testing.T) {
	mux, closeRepo := testRouter(t)
	defer closeRepo()

	body := []byte(`{"event_id":"evt-1","tenant_id":"tenant-b","worker_id":"worker-a","provider":"demo","model_id":"efficient-model","prompt_tokens":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ingest/token-usage", bytes.NewReader(body))
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Error == nil || envelope.Error.Code != "tenant_mismatch" {
		t.Fatalf("expected tenant mismatch envelope, got %#v", envelope)
	}
}

func TestDashboardReadDegradesWhenDatabaseUnavailable(t *testing.T) {
	repo := storage.NewUnavailableRepository(storage.ErrUnavailable)
	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	mux := NewRouter(service, repo, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 degraded read, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.OK || envelope.Status != "degraded" {
		t.Fatalf("expected degraded ok envelope, got %#v", envelope)
	}
}

func TestHandleExportCSV(t *testing.T) {
	router, cleanup := testRouter(t)
	defer cleanup()

	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/export.csv", nil)
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/csv" {
		t.Fatalf("expected content type text/csv, got %s", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()
	if len(body) == 0 {
		t.Fatalf("expected non-empty CSV body")
	}
}

func TestTokenUsageRouteIngestsStructuredPayload(t *testing.T) {
	mux, closeRepo := testRouter(t)
	defer closeRepo()

	body := []byte(`{"event_id":"evt-2","worker_id":"worker-a","provider":"demo","model_id":"efficient-model","prompt_tokens":1000,"completion_tokens":100,"output_status":"accepted","review_score":88}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ingest/token-usage", bytes.NewReader(body))
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.OK || envelope.Status != "degraded" {
		t.Fatalf("expected degraded envelope with buffered, got %#v", envelope)
	}
	hasBuffered := false
	for _, issue := range envelope.Degraded {
		if issue.Code == "buffered" {
			hasBuffered = true
			break
		}
	}
	if !hasBuffered {
		t.Fatalf("expected buffered degradation issue, got %#v", envelope.Degraded)
	}
}

func TestNewEndpoints(t *testing.T) {
	mux, closeRepo := testRouter(t)
	defer closeRepo()

	// 1. Test GET /api/pricing
	req := httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode pricing: %v", err)
	}
	if !envelope.OK || envelope.Status != "success" {
		t.Fatalf("expected ok envelope, got %#v", envelope)
	}

	// 2. Test POST /api/dashboard/seed
	reqSeed := httptest.NewRequest(http.MethodPost, "/api/dashboard/seed", nil)
	reqSeed.Header.Set("x-tenant-id", "tenant-a")
	recSeed := httptest.NewRecorder()
	mux.ServeHTTP(recSeed, reqSeed)

	if recSeed.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recSeed.Code, recSeed.Body.String())
	}

	// 3. Test DELETE /api/dashboard/reset
	reqReset := httptest.NewRequest(http.MethodDelete, "/api/dashboard/reset", nil)
	reqReset.Header.Set("x-tenant-id", "tenant-a")
	recReset := httptest.NewRecorder()
	mux.ServeHTTP(recReset, reqReset)

	if recReset.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recReset.Code, recReset.Body.String())
	}
}

func TestWorkerReviewAndReportEndpoints(t *testing.T) {
	mux, closeRepo := testRouter(t)
	defer closeRepo()

	body := []byte(`{"event_id":"evt-review","worker_id":"worker-review","provider":"demo","model_id":"efficient-model","input_tokens":100,"output_tokens":500,"prompt_excerpt":"Write an answer.","output_excerpt":"This repeats without evidence. This repeats without evidence.","output_status":"accepted"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ingest/token-usage", bytes.NewReader(body))
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected ingest 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	time.Sleep(150 * time.Millisecond)

	reviewReq := httptest.NewRequest(http.MethodGet, "/api/dashboard/workers/worker-review", nil)
	reviewReq.Header.Set("x-tenant-id", "tenant-a")
	reviewRec := httptest.NewRecorder()
	mux.ServeHTTP(reviewRec, reviewReq)
	if reviewRec.Code != http.StatusOK {
		t.Fatalf("expected review 200, got %d body=%s", reviewRec.Code, reviewRec.Body.String())
	}

	reportReq := httptest.NewRequest(http.MethodGet, "/api/dashboard/report.md", nil)
	reportReq.Header.Set("x-tenant-id", "tenant-a")
	reportRec := httptest.NewRecorder()
	mux.ServeHTTP(reportRec, reportReq)
	if reportRec.Code != http.StatusOK {
		t.Fatalf("expected report 200, got %d body=%s", reportRec.Code, reportRec.Body.String())
	}
	if reportRec.Header().Get("Content-Type") != "text/markdown; charset=utf-8" {
		t.Fatalf("expected markdown content type, got %s", reportRec.Header().Get("Content-Type"))
	}
}

func TestRecommendationStateTenantMembersAndAudit(t *testing.T) {
	mux, closeRepo := testRouter(t)
	defer closeRepo()

	updateBody := []byte(`{"status":"accepted","note":"Use for low-risk classification."}`)
	stateReq := httptest.NewRequest(http.MethodPost, "/api/dashboard/recommendations/rec_test/status", bytes.NewReader(updateBody))
	stateReq.Header.Set("x-tenant-id", "tenant-a")
	stateRec := httptest.NewRecorder()
	mux.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("expected recommendation state 200, got %d body=%s", stateRec.Code, stateRec.Body.String())
	}

	memberBody := []byte(`{"subject_id":"user_1","email":"ops@example.com","role":"analyst"}`)
	memberReq := httptest.NewRequest(http.MethodPost, "/api/tenant/members", bytes.NewReader(memberBody))
	memberReq.Header.Set("x-tenant-id", "tenant-a")
	memberRec := httptest.NewRecorder()
	mux.ServeHTTP(memberRec, memberReq)
	if memberRec.Code != http.StatusOK {
		t.Fatalf("expected member upsert 200, got %d body=%s", memberRec.Code, memberRec.Body.String())
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/api/audit/events", nil)
	auditReq.Header.Set("x-tenant-id", "tenant-a")
	auditRec := httptest.NewRecorder()
	mux.ServeHTTP(auditRec, auditReq)
	if auditRec.Code != http.StatusOK {
		t.Fatalf("expected audit 200, got %d body=%s", auditRec.Code, auditRec.Body.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(auditRec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	if envelope.Status != "success" {
		t.Fatalf("expected audit success with persisted events, got %#v", envelope)
	}
}

func TestViewerAPIKeyCannotMutateTenant(t *testing.T) {
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// Seed tenant-a
	tenant := domain.Tenant{
		TenantID:  "tenant-a",
		Name:      "Tenant A",
		Tier:      "free",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpsertTenant(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant-a: %v", err)
	}

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	apiKey, token, err := moat.GenerateAPIKey("tenant-a", "viewer")
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	apiKey.Role = domain.RoleViewer
	if err := repo.SaveAPIKey(context.Background(), apiKey); err != nil {
		t.Fatalf("save key: %v", err)
	}
	mux := NewRouter(service, repo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/dashboard/reset", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func testRouter(t *testing.T) (http.Handler, func()) {
	t.Helper()
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Seed tenant-a
	tenant := domain.Tenant{
		TenantID:  "tenant-a",
		Name:      "Tenant A",
		Tier:      "free",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpsertTenant(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant-a: %v", err)
	}

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{})).WithClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	service.StartWorker(context.Background())
	return NewRouter(service, repo, nil), func() { _ = repo.Close() }
}

func TestStripeWebhookHandler(t *testing.T) {
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// Seed target tenant
	tenant := domain.Tenant{
		TenantID:         "tenant-stripe-test",
		Name:             "Stripe Test Tenant",
		Tier:             "free",
		UsageLimitUSD:    10.0,
		StripeCustomerID: "cus_stripe_123",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := repo.UpsertTenant(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	mux := NewRouter(service, repo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader([]byte(`{"type":"customer.subscription.created"}`)))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := repo.GetTenant(context.Background(), "tenant-stripe-test")
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if updated.Tier != "free" || updated.StripeSubscriptionID != "" {
		t.Fatalf("Go webhook route should not mutate billing state: tier=%s subscription=%s", updated.Tier, updated.StripeSubscriptionID)
	}
}

func TestVerifiedStripeEventRouteAppliesBillingLifecycle(t *testing.T) {
	t.Setenv("TG_INTERNAL_WEBHOOK_SECRET", "internal-secret")

	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	now := time.Now().UTC()
	if err := repo.UpsertTenant(context.Background(), domain.Tenant{
		TenantID:         "tenant-stripe-test",
		Name:             "Stripe Test Tenant",
		Tier:             "free",
		UsageLimitUSD:    10,
		StripeCustomerID: "cus_stripe_123",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	mux := NewRouter(service, repo, nil)

	payload := []byte(`{
		"event_id": "evt_123",
		"event_type": "customer.subscription.updated",
		"customer_id": "cus_stripe_123",
		"subscription_id": "sub_stripe_abc",
		"subscription_status": "active"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/billing/stripe-event", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer internal-secret")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	updated, err := repo.GetTenant(context.Background(), "tenant-stripe-test")
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if updated.Tier != "premium" || updated.UsageLimitUSD != 100 || updated.StripeSubscriptionID != "sub_stripe_abc" {
		t.Fatalf("billing lifecycle not applied: %+v", updated)
	}
}
