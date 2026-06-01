package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	_ "modernc.org/sqlite"
)

func TestOpenSQLiteRepairsOlderSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "old.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw sqlite: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE tenants (
			tenant_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE workers (
			tenant_id TEXT NOT NULL,
			worker_id TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, worker_id)
		);
		CREATE TABLE token_usage_events (
			tenant_id TEXT NOT NULL,
			event_id TEXT NOT NULL,
			worker_id TEXT NOT NULL,
			worker_name TEXT NOT NULL,
			job_id TEXT,
			session_id TEXT,
			run_id TEXT,
			provider TEXT NOT NULL,
			model_id TEXT NOT NULL,
			prompt_tokens INTEGER NOT NULL,
			completion_tokens INTEGER NOT NULL,
			cached_tokens INTEGER NOT NULL,
			input_tokens INTEGER NOT NULL,
			output_tokens INTEGER NOT NULL,
			total_tokens INTEGER NOT NULL,
			cost_estimate_usd REAL,
			cost_currency TEXT NOT NULL,
			cost_is_degraded INTEGER NOT NULL,
			cost_degraded_code TEXT,
			external_estimate_usd REAL,
			external_estimate_currency TEXT,
			latency_ms INTEGER NOT NULL,
			task_category TEXT NOT NULL,
			output_status TEXT NOT NULL,
			review_score REAL,
			occurred_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			tags_json TEXT,
			PRIMARY KEY (tenant_id, event_id)
		);
	`)
	if err != nil {
		t.Fatalf("create old schema: %v", err)
	}
	_ = db.Close()

	repo, err := OpenSQLite(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open repaired sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if err := repo.SaveTokenEvent(context.Background(), domain.TokenEvent{
		TenantID:       "tenant-a",
		EventID:        "evt-1",
		WorkerID:       "worker-a",
		WorkerName:     "Worker A",
		Provider:       "demo",
		ModelID:        "efficient-model",
		InputTokens:    10,
		OutputTokens:   5,
		TotalTokens:    15,
		CostCurrency:   "USD",
		TaskCategory:   "test",
		OutputStatus:   domain.OutputAccepted,
		Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		OutputExcerpt:  "Verified concise output.",
		IdempotencyKey: "idem-1",
	}); err != nil {
		t.Fatalf("save after repair: %v", err)
	}
}
