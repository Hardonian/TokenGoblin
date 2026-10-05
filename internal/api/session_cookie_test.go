package api

import (
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

// The frontend keeps its session in http-only cookies (tg_api_key /
// tg_tenant_id) and calls proxied routes WITHOUT an Authorization header.
// The middleware must treat those cookies as identity — otherwise the whole
// dashboard reads 401 tenant_missing after login.
func TestSessionCookieAuthenticatesAPIRequests(t *testing.T) {
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	tenant := domain.Tenant{
		TenantID:  "tenant-cookie",
		Name:      "Cookie Tenant",
		Tier:      "free",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpsertTenant(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	apiKey, token, err := moat.GenerateAPIKey("tenant-cookie", "owner")
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	if err := repo.SaveAPIKey(context.Background(), apiKey); err != nil {
		t.Fatalf("save key: %v", err)
	}

	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	service.StartWorker(context.Background())
	mux := NewRouter(service, repo, nil)

	// Cookie only — no Authorization header, like the real frontend.
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	req.AddCookie(&http.Cookie{Name: "tg_api_key", Value: token})
	req.AddCookie(&http.Cookie{Name: "tg_tenant_id", Value: "tenant-cookie"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("cookie session: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// The analytics route proves AUTHENTICATION specifically: without a
	// mirror it answers 503 analytics_unavailable once authenticated — a 401
	// here would mean the cookie identity was ignored.
	req1b := httptest.NewRequest(http.MethodGet, "/v1/analytics/cost", nil)
	req1b.AddCookie(&http.Cookie{Name: "tg_api_key", Value: token})
	rec1b := httptest.NewRecorder()
	mux.ServeHTTP(rec1b, req1b)
	if rec1b.Code != http.StatusServiceUnavailable {
		t.Fatalf("cookie auth: expected 503 analytics_unavailable (auth passed), got %d body=%s",
			rec1b.Code, rec1b.Body.String())
	}
	var authEnvelope Envelope
	if err := json.Unmarshal(rec1b.Body.Bytes(), &authEnvelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if authEnvelope.Error == nil || authEnvelope.Error.Code != "analytics_unavailable" {
		t.Fatalf("cookie auth: expected analytics_unavailable (means auth PASSED), got %#v", authEnvelope)
	}

	// Tenant cookie alone (demo mode) must also resolve — and must resolve to
	// the cookie's tenant, not any client-controlled header.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	req2.AddCookie(&http.Cookie{Name: "tg_tenant_id", Value: "tenant-cookie"})
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("tenant cookie: expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}
