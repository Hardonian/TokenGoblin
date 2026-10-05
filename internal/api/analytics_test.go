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
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/Hardonian/TokenGoblin/internal/storage/clickhouse"
)

type fakeAnalytics struct {
	lastTenant string
	calls      int
	fail       bool
}

func (f *fakeAnalytics) GetCostByTenant(ctx context.Context, tenantID string, start, end time.Time) (clickhouse.CostSummary, error) {
	f.calls++
	f.lastTenant = tenantID
	if f.fail {
		return clickhouse.CostSummary{}, &clickhouse.QueryError{Op: "test", Err: context.DeadlineExceeded}
	}
	return clickhouse.CostSummary{TenantID: tenantID, TotalCostUSD: 12.5, TotalTokens: 100, RequestCount: 7}, nil
}

func (f *fakeAnalytics) GetCostByModel(ctx context.Context, tenantID string, start, end time.Time) ([]clickhouse.ModelCostSummary, error) {
	f.calls++
	f.lastTenant = tenantID
	return []clickhouse.ModelCostSummary{{Model: "model-x", TotalCostUSD: 3.5, RequestCount: 2}}, nil
}

func (f *fakeAnalytics) GetCostByFeature(ctx context.Context, tenantID string, start, end time.Time) ([]clickhouse.FeatureCostSummary, error) {
	f.calls++
	f.lastTenant = tenantID
	return []clickhouse.FeatureCostSummary{{Feature: "research", TotalCostUSD: 9, ROI: 1.5}}, nil
}

func (f *fakeAnalytics) GetZombieAgents(ctx context.Context, tenantID string, threshold float64, window time.Duration) ([]clickhouse.ZombieAgentRecord, error) {
	f.calls++
	f.lastTenant = tenantID
	return []clickhouse.ZombieAgentRecord{{AgentID: "worker-z", TenantID: tenantID, AcceptanceRate: 0, TotalCost: 4, Recommendation: "quarantine"}}, nil
}

func analyticsTestRouter(t *testing.T, analytics AnalyticsStore, tier string) (http.Handler, func()) {
	t.Helper()
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	tenant := domain.Tenant{
		TenantID:  "tenant-a",
		Name:      "Tenant A",
		Tier:      tier,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpsertTenant(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	service.StartWorker(context.Background())
	var opts []RouterOption
	if analytics != nil {
		opts = append(opts, WithAnalytics(analytics))
	}
	return NewRouter(service, repo, nil, opts...), func() { _ = repo.Close() }
}

func TestAnalyticsUnavailableWithoutMirror(t *testing.T) {
	mux, closeRepo := analyticsTestRouter(t, nil, "enterprise")
	defer closeRepo()

	req := httptest.NewRequest(http.MethodGet, "/v1/analytics/cost", nil)
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when mirror unconfigured, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envelope.Error == nil || envelope.Error.Code != "analytics_unavailable" {
		t.Fatalf("expected analytics_unavailable error, got %#v", envelope)
	}
}

func TestAnalyticsQueriesAreTenantScoped(t *testing.T) {
	fake := &fakeAnalytics{}
	mux, closeRepo := analyticsTestRouter(t, fake, "free")
	defer closeRepo()

	for _, path := range []string{"/v1/analytics/cost", "/v1/analytics/cost/by-model", "/v1/analytics/cost/by-feature"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("x-tenant-id", "tenant-a")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d body=%s", path, rec.Code, rec.Body.String())
		}
		if fake.lastTenant != "tenant-a" {
			t.Fatalf("%s: analytics queried tenant %q, want the AUTHENTICATED tenant", path, fake.lastTenant)
		}
	}
}

func TestAnalyticsQueryFailureDegradesHonest(t *testing.T) {
	fake := &fakeAnalytics{fail: true}
	mux, closeRepo := analyticsTestRouter(t, fake, "free")
	defer closeRepo()

	req := httptest.NewRequest(http.MethodGet, "/v1/analytics/cost", nil)
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 on query failure, got %d", rec.Code)
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envelope.Error == nil || envelope.Error.Code != "analytics_query_failed" {
		t.Fatalf("expected analytics_query_failed, got %#v", envelope)
	}
}

func TestZombieAgentsRequiresPaidTier(t *testing.T) {
	// Free tier: the signature feature must be refused.
	fake := &fakeAnalytics{}
	mux, closeRepo := analyticsTestRouter(t, fake, "free")
	req := httptest.NewRequest(http.MethodGet, "/v1/analytics/zombie-agents", nil)
	req.Header.Set("x-tenant-id", "tenant-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("free tier: expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	if fake.calls != 0 {
		t.Fatal("free tier must never reach the analytics store")
	}
	closeRepo()

	// Paid tier: served.
	fake2 := &fakeAnalytics{}
	mux2, closeRepo2 := analyticsTestRouter(t, fake2, "enterprise")
	defer closeRepo2()
	req2 := httptest.NewRequest(http.MethodGet, "/v1/analytics/zombie-agents?threshold=0.5&window_hours=48", nil)
	req2.Header.Set("x-tenant-id", "tenant-a")
	rec2 := httptest.NewRecorder()
	mux2.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("enterprise tier: expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	if fake2.lastTenant != "tenant-a" {
		t.Fatalf("zombie agents queried tenant %q, want tenant-a", fake2.lastTenant)
	}
}