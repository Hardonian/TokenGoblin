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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTwoTenants(t *testing.T) (http.Handler, storage.Repository, string, string, string, string) {
	t.Helper()
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "tenant_isolation.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	now := time.Now().UTC()

	// Seed Tenant A
	tenantA := domain.Tenant{
		TenantID:      "tenant-alpha",
		Name:          "Alpha Corp",
		Tier:          "pro",
		UsageLimitUSD: 100.0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repo.UpsertTenant(context.Background(), tenantA))

	// Seed Tenant B
	tenantB := domain.Tenant{
		TenantID:      "tenant-beta",
		Name:          "Beta Corp",
		Tier:          "enterprise",
		UsageLimitUSD: 1000.0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repo.UpsertTenant(context.Background(), tenantB))

	// Generate and save keys for Tenant A and Tenant B
	keyA, tokenA, err := moat.GenerateAPIKey("tenant-alpha", "admin-key-a")
	require.NoError(t, err)
	keyA.Role = domain.RoleAdmin
	require.NoError(t, repo.SaveAPIKey(context.Background(), keyA))

	keyB, tokenB, err := moat.GenerateAPIKey("tenant-beta", "admin-key-b")
	require.NoError(t, err)
	keyB.Role = domain.RoleAdmin
	require.NoError(t, repo.SaveAPIKey(context.Background(), keyB))

	// Add distinct events for Tenant A and B
	costA := 12.50
	costB := 75.00
	require.NoError(t, repo.SaveTokenEvent(context.Background(), domain.TokenEvent{
		EventID:         "evt-a-1",
		TenantID:        "tenant-alpha",
		WorkerID:        "worker-alpha-1",
		Provider:        "openai",
		ModelID:         "gpt-4o",
		PromptTokens:    500,
		TotalTokens:     600,
		CostEstimateUSD: &costA,
		CostCurrency:    "USD",
		Timestamp:       now,
	}))

	require.NoError(t, repo.SaveTokenEvent(context.Background(), domain.TokenEvent{
		EventID:         "evt-b-1",
		TenantID:        "tenant-beta",
		WorkerID:        "worker-beta-1",
		Provider:        "anthropic",
		ModelID:         "claude-3-5-sonnet",
		PromptTokens:    2000,
		TotalTokens:     2500,
		CostEstimateUSD: &costB,
		CostCurrency:    "USD",
		Timestamp:       now,
	}))

	// Audit events
	require.NoError(t, repo.SaveAuditEvent(context.Background(), domain.AuditEvent{
		EventID:   "aud-a-1",
		TenantID:  "tenant-alpha",
		Type:      "secret.accessed",
		Actor:     "tenant:tenant-alpha",
		Resource:  "vault:secret-alpha",
		Timestamp: now,
	}))
	require.NoError(t, repo.SaveAuditEvent(context.Background(), domain.AuditEvent{
		EventID:   "aud-b-1",
		TenantID:  "tenant-beta",
		Type:      "secret.accessed",
		Actor:     "tenant:tenant-beta",
		Resource:  "vault:secret-beta",
		Timestamp: now,
	}))

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	router := NewRouter(service, repo, nil)

	return router, repo, tokenA, tokenB, keyA.KeyID, keyB.KeyID
}

func TestTenantCannotReadAnotherTenantsKeys(t *testing.T) {
	router, _, tokenA, _, keyA_ID, keyB_ID := setupTwoTenants(t)

	// Tenant A requests keys
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/keys", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.True(t, env.OK)

	dataJSON, err := json.Marshal(env.Data)
	require.NoError(t, err)

	var keys []struct {
		KeyID string `json:"key_id"`
		Name  string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(dataJSON, &keys))

	// Tenant A should see only key A, NOT key B
	require.Len(t, keys, 1)
	assert.Equal(t, keyA_ID, keys[0].KeyID)
	assert.NotEqual(t, keyB_ID, keys[0].KeyID)
}

func TestTenantCannotRevokeAnotherTenantsKey(t *testing.T) {
	router, repo, tokenA, _, _, keyB_ID := setupTwoTenants(t)

	// Tenant A attempts to revoke Tenant B's key
	req := httptest.NewRequest(http.MethodDelete, "/api/tenant/keys?key_id="+keyB_ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Check Tenant B's key in repo: must NOT be revoked!
	keyB, err := repo.GetAPIKey(context.Background(), keyB_ID)
	require.NoError(t, err)
	assert.False(t, keyB.IsRevoked, "Tenant B's key must not be revoked by Tenant A")
}

func TestTenantCannotReadAnotherTenantsSpend(t *testing.T) {
	router, _, tokenA, tokenB, _, _ := setupTwoTenants(t)

	// Tenant A billing status
	reqA := httptest.NewRequest(http.MethodGet, "/api/billing/status", nil)
	reqA.Header.Set("Authorization", "Bearer "+tokenA)
	recA := httptest.NewRecorder()
	router.ServeHTTP(recA, reqA)
	assert.Equal(t, http.StatusOK, recA.Code)

	var envA struct {
		Data struct {
			TenantID            string  `json:"tenant_id"`
			Tier                string  `json:"tier"`
			CurrentMonthCostUSD float64 `json:"current_month_cost_usd"`
			UsageLimitUSD       float64 `json:"usage_limit_usd"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recA.Body.Bytes(), &envA))
	assert.Equal(t, "tenant-alpha", envA.Data.TenantID)
	assert.Equal(t, "pro", envA.Data.Tier)
	assert.InDelta(t, 12.50, envA.Data.CurrentMonthCostUSD, 0.01)

	// Tenant B billing status
	reqB := httptest.NewRequest(http.MethodGet, "/api/billing/status", nil)
	reqB.Header.Set("Authorization", "Bearer "+tokenB)
	recB := httptest.NewRecorder()
	router.ServeHTTP(recB, reqB)
	assert.Equal(t, http.StatusOK, recB.Code)

	var envB struct {
		Data struct {
			TenantID            string  `json:"tenant_id"`
			Tier                string  `json:"tier"`
			CurrentMonthCostUSD float64 `json:"current_month_cost_usd"`
			UsageLimitUSD       float64 `json:"usage_limit_usd"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &envB))
	assert.Equal(t, "tenant-beta", envB.Data.TenantID)
	assert.Equal(t, "enterprise", envB.Data.Tier)
	assert.InDelta(t, 75.00, envB.Data.CurrentMonthCostUSD, 0.01)
}

func TestTenantCannotReadAnotherTenantsAuditLogs(t *testing.T) {
	router, _, tokenA, _, _, _ := setupTwoTenants(t)

	// Tenant A requests audit logs
	req := httptest.NewRequest(http.MethodGet, "/api/audit/events", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "success", env.Status)

	dataJSON, err := json.Marshal(env.Data)
	require.NoError(t, err)

	var events []domain.AuditEvent
	require.NoError(t, json.Unmarshal(dataJSON, &events))

	// Tenant A must only see aud-a-1, never aud-b-1
	for _, ev := range events {
		assert.Equal(t, "tenant-alpha", ev.TenantID)
		assert.NotEqual(t, "tenant-beta", ev.TenantID)
		assert.NotEqual(t, "aud-b-1", ev.EventID)
	}
}

func TestTenantCannotResetAnotherTenantsData(t *testing.T) {
	router, repo, tokenA, _, _, _ := setupTwoTenants(t)

	// Tenant A attempts to reset data (which should only clear tenant A's data)
	req := httptest.NewRequest(http.MethodDelete, "/api/dashboard/reset", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Check that Tenant B's events are intact
	eventsB, err := repo.ListTokenEvents(context.Background(), "tenant-beta", 10)
	require.NoError(t, err)
	assert.Len(t, eventsB, 1, "Tenant B's events must remain intact after Tenant A reset")

	// Check that Tenant A's events were cleared
	eventsA, err := repo.ListTokenEvents(context.Background(), "tenant-alpha", 10)
	require.NoError(t, err)
	assert.Len(t, eventsA, 0, "Tenant A's events should be cleared")
}

func TestTenantCannotIngestDataForAnotherTenant(t *testing.T) {
	router, repo, tokenA, _, _, _ := setupTwoTenants(t)

	// Tenant A sends an event with payload tenant_id: "tenant-beta"
	body := []byte(`{
		"event_id": "evt-spoofed",
		"tenant_id": "tenant-beta",
		"worker_id": "worker-spoofed",
		"provider": "openai",
		"model_id": "gpt-4o",
		"prompt_tokens": 100,
		"completion_tokens": 100
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ingest/token-usage", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Ingestion service should reject cross-tenant spoofing with 403 Forbidden
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Confirm event was not written to Tenant B
	eventsB, err := repo.ListTokenEvents(context.Background(), "tenant-beta", 10)
	require.NoError(t, err)
	for _, ev := range eventsB {
		assert.NotEqual(t, "evt-spoofed", ev.EventID)
	}
}
