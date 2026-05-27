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
	service := ingestion.NewService(storage.NewUnavailableRepository(storage.ErrUnavailable), cost.LoadRegistry(context.Background(), cost.RegistryConfig{}))
	mux := NewRouter(service)
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

func testRouter(t *testing.T) (*http.ServeMux, func()) {
	t.Helper()
	repo, err := storage.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	service := ingestion.NewService(repo, cost.LoadRegistry(context.Background(), cost.RegistryConfig{})).WithClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	service.StartWorker(context.Background())
	return NewRouter(service), func() { _ = repo.Close() }
}
