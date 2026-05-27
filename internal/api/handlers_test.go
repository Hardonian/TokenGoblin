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
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
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

func testRouter(t *testing.T) (http.Handler, func()) {
	t.Helper()
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{})).WithClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	service.StartWorker(context.Background())
	return NewRouter(service, repo, nil), func() { _ = repo.Close() }
}
