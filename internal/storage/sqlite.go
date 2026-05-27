package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	_ "modernc.org/sqlite"
)

const defaultDBPath = "./data/tokengoblin.sqlite"

type SQLiteRepository struct {
	db *sql.DB
}

func OpenFromEnv(ctx context.Context) (Repository, error) {
	path := os.Getenv("TG_DB_PATH")
	if path == "" {
		path = defaultDBPath
	}
	return OpenSQLite(ctx, path)
}

func OpenSQLite(ctx context.Context, dbPath string) (*SQLiteRepository, error) {
	if dbPath == "" {
		dbPath = defaultDBPath
	}
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, fmt.Errorf("%w: create database directory: %v", ErrUnavailable, err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: open sqlite: %v", ErrUnavailable, err)
	}
	db.SetMaxOpenConns(1)

	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS tenants (
			tenant_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS workers (
			tenant_id TEXT NOT NULL,
			worker_id TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, worker_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS jobs (
			tenant_id TEXT NOT NULL,
			job_id TEXT NOT NULL,
			worker_id TEXT NOT NULL,
			name TEXT NOT NULL,
			task_category TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at TEXT,
			ended_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, job_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			FOREIGN KEY (tenant_id, worker_id) REFERENCES workers(tenant_id, worker_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS token_usage_events (
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
			idempotency_key TEXT,
			PRIMARY KEY (tenant_id, event_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			FOREIGN KEY (tenant_id, worker_id) REFERENCES workers(tenant_id, worker_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_token_usage_tenant_occurred ON token_usage_events (tenant_id, occurred_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_token_usage_tenant_worker ON token_usage_events (tenant_id, worker_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_token_usage_idempotency ON token_usage_events(tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			key_id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT,
			is_revoked INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);`,
		`CREATE TABLE IF NOT EXISTS cost_snapshots (
			tenant_id TEXT NOT NULL,
			snapshot_id TEXT NOT NULL,
			event_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			model_id TEXT NOT NULL,
			input_tokens INTEGER NOT NULL,
			output_tokens INTEGER NOT NULL,
			cached_tokens INTEGER NOT NULL,
			cost_estimate_usd REAL,
			currency TEXT NOT NULL,
			is_degraded INTEGER NOT NULL,
			degraded_code TEXT,
			created_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, snapshot_id),
			FOREIGN KEY (tenant_id, event_id) REFERENCES token_usage_events(tenant_id, event_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS anomaly_signals (
			tenant_id TEXT NOT NULL,
			anomaly_id TEXT NOT NULL,
			event_id TEXT,
			worker_id TEXT,
			detected_at TEXT NOT NULL,
			severity TEXT NOT NULL,
			type TEXT NOT NULL,
			description TEXT NOT NULL,
			observed_value REAL,
			threshold_value REAL,
			details_json TEXT,
			created_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, anomaly_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_anomaly_tenant_detected ON anomaly_signals (tenant_id, detected_at DESC);`,
		`CREATE TABLE IF NOT EXISTS productivity_summaries (
			tenant_id TEXT NOT NULL,
			summary_id TEXT NOT NULL,
			period_start TEXT,
			period_end TEXT,
			generated_at TEXT NOT NULL,
			total_cost_usd REAL NOT NULL,
			total_events INTEGER NOT NULL,
			output_count INTEGER NOT NULL,
			avg_latency_ms REAL,
			anomaly_count INTEGER NOT NULL,
			cost_per_accepted_output_with_review REAL,
			summary_json TEXT NOT NULL,
			PRIMARY KEY (tenant_id, summary_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
	}

	for _, statement := range statements {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("%w: migrate sqlite: %v", ErrUnavailable, err)
		}
	}
	return nil
}

func (r *SQLiteRepository) UpsertTenant(ctx context.Context, tenant domain.Tenant) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tenants (tenant_id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(tenant_id) DO UPDATE SET
			name = excluded.name,
			updated_at = excluded.updated_at
	`, tenant.TenantID, tenant.Name, formatTime(tenant.CreatedAt), formatTime(tenant.UpdatedAt))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) SaveAPIKey(ctx context.Context, key domain.APIKey) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO api_keys (key_id, tenant_id, name, key_hash, created_at, last_used_at, is_revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, key.KeyID, key.TenantID, key.Name, key.KeyHash, formatTime(key.CreatedAt), timePtrString(key.LastUsedAt), boolInt(key.IsRevoked))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error) {
	var key domain.APIKey
	var createdAt string
	var lastUsedAt sql.NullString
	var isRevoked int
	err := r.db.QueryRowContext(ctx, `
		SELECT key_id, tenant_id, name, key_hash, created_at, last_used_at, is_revoked
		FROM api_keys
		WHERE key_id = ?
	`, keyID).Scan(&key.KeyID, &key.TenantID, &key.Name, &key.KeyHash, &createdAt, &lastUsedAt, &isRevoked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	key.CreatedAt = parseTime(createdAt)
	if lastUsedAt.Valid {
		parsed := parseTime(lastUsedAt.String)
		key.LastUsedAt = &parsed
	}
	key.IsRevoked = isRevoked == 1
	return &key, nil
}

func (r *SQLiteRepository) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_keys
		SET last_used_at = ?
		WHERE key_id = ?
	`, formatTime(time.Now().UTC()), keyID)
	return wrapDBErr(err)
}

func (r *SQLiteRepository) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return wrapDBErr(err)
	}
	defer rollback(tx)

	now := event.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	tenant := domain.Tenant{
		TenantID:  event.TenantID,
		Name:      event.TenantID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tenants (tenant_id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(tenant_id) DO UPDATE SET updated_at = excluded.updated_at
	`, tenant.TenantID, tenant.Name, formatTime(tenant.CreatedAt), formatTime(tenant.UpdatedAt)); err != nil {
		return wrapDBErr(err)
	}

	workerName := event.WorkerName
	if workerName == "" {
		workerName = event.WorkerID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workers (tenant_id, worker_id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, worker_id) DO UPDATE SET
			name = excluded.name,
			updated_at = excluded.updated_at
	`, event.TenantID, event.WorkerID, workerName, formatTime(now), formatTime(now)); err != nil {
		return wrapDBErr(err)
	}

	taskCategory := event.TaskCategory
	if taskCategory == "" {
		taskCategory = event.TaskType
	}
	if taskCategory == "" {
		taskCategory = "uncategorized"
	}
	if event.JobID != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO jobs (tenant_id, job_id, worker_id, name, task_category, status, started_at, ended_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(tenant_id, job_id) DO UPDATE SET
				worker_id = excluded.worker_id,
				task_category = excluded.task_category,
				updated_at = excluded.updated_at
		`, event.TenantID, event.JobID, event.WorkerID, event.JobID, taskCategory, "active", nil, nil, formatTime(now), formatTime(now)); err != nil {
			return wrapDBErr(err)
		}
	}

	tagsJSON, err := marshalNullable(event.Tags)
	if err != nil {
		return err
	}

	var externalCost interface{}
	var externalCurrency interface{}
	if event.ExternalEstimate != nil {
		externalCost = event.ExternalEstimate.CostUSD
		externalCurrency = event.ExternalEstimate.Currency
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO token_usage_events (
			tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
			provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
			input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
			cost_is_degraded, cost_degraded_code, external_estimate_usd,
			external_estimate_currency, latency_ms, task_category, output_status,
			review_score, occurred_at, created_at, tags_json, idempotency_key
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, event_id) DO UPDATE SET
			worker_id = excluded.worker_id,
			worker_name = excluded.worker_name,
			job_id = excluded.job_id,
			session_id = excluded.session_id,
			run_id = excluded.run_id,
			provider = excluded.provider,
			model_id = excluded.model_id,
			prompt_tokens = excluded.prompt_tokens,
			completion_tokens = excluded.completion_tokens,
			cached_tokens = excluded.cached_tokens,
			input_tokens = excluded.input_tokens,
			output_tokens = excluded.output_tokens,
			total_tokens = excluded.total_tokens,
			cost_estimate_usd = excluded.cost_estimate_usd,
			cost_currency = excluded.cost_currency,
			cost_is_degraded = excluded.cost_is_degraded,
			cost_degraded_code = excluded.cost_degraded_code,
			external_estimate_usd = excluded.external_estimate_usd,
			external_estimate_currency = excluded.external_estimate_currency,
			latency_ms = excluded.latency_ms,
			task_category = excluded.task_category,
			output_status = excluded.output_status,
			review_score = excluded.review_score,
			occurred_at = excluded.occurred_at,
			tags_json = excluded.tags_json,
			idempotency_key = excluded.idempotency_key
	`, event.TenantID, event.EventID, event.WorkerID, workerName, nullString(event.JobID),
		nullString(event.SessionID), nullString(event.RunID), event.Provider, event.ModelID,
		event.PromptTokens, event.CompletionTokens, event.CachedTokens, event.InputTokens,
		event.OutputTokens, event.TotalTokens, event.CostEstimateUSD, event.CostCurrency,
		boolInt(event.CostIsDegraded), nullString(event.CostDegradedCode), externalCost,
		externalCurrency, event.LatencyMs, taskCategory, string(event.OutputStatus),
		event.ReviewScore, formatTime(event.Timestamp), formatTime(now), tagsJSON, nullString(event.IdempotencyKey))
	if err != nil {
		return wrapDBErr(err)
	}

	return wrapDBErr(tx.Commit())
}

func (r *SQLiteRepository) SaveCostSnapshot(ctx context.Context, snapshot domain.CostSnapshot) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cost_snapshots (
			tenant_id, snapshot_id, event_id, provider, model_id, input_tokens,
			output_tokens, cached_tokens, cost_estimate_usd, currency, is_degraded,
			degraded_code, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, snapshot_id) DO UPDATE SET
			cost_estimate_usd = excluded.cost_estimate_usd,
			is_degraded = excluded.is_degraded,
			degraded_code = excluded.degraded_code,
			created_at = excluded.created_at
	`, snapshot.TenantID, snapshot.SnapshotID, snapshot.EventID, snapshot.Provider,
		snapshot.ModelID, snapshot.InputTokens, snapshot.OutputTokens, snapshot.CachedTokens,
		snapshot.CostEstimateUSD, snapshot.Currency, boolInt(snapshot.IsDegraded),
		nullString(snapshot.DegradedCode), formatTime(snapshot.CreatedAt))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) SaveAnomalySignal(ctx context.Context, signal domain.AnomalySignal) error {
	detailsJSON, err := marshalNullable(signal.Details)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO anomaly_signals (
			tenant_id, anomaly_id, event_id, worker_id, detected_at, severity,
			type, description, observed_value, threshold_value, details_json, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, anomaly_id) DO UPDATE SET
			detected_at = excluded.detected_at,
			severity = excluded.severity,
			description = excluded.description,
			observed_value = excluded.observed_value,
			threshold_value = excluded.threshold_value,
			details_json = excluded.details_json,
			created_at = excluded.created_at
	`, signal.TenantID, signal.AnomalyID, nullString(signal.EventID), nullString(signal.WorkerID),
		formatTime(signal.DetectedAt), string(signal.Severity), string(signal.Type),
		signal.Description, signal.ObservedValue, signal.ThresholdValue, detailsJSON,
		formatTime(signal.DetectedAt))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) SaveProductivitySummary(ctx context.Context, summary domain.ProductivitySummary) error {
	body, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	var avg interface{}
	if summary.AvgLatencyMs != nil {
		avg = *summary.AvgLatencyMs
	}
	var costPerAccepted interface{}
	if summary.CostPerAcceptedOutputWithReview != nil {
		costPerAccepted = *summary.CostPerAcceptedOutputWithReview
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO productivity_summaries (
			tenant_id, summary_id, period_start, period_end, generated_at,
			total_cost_usd, total_events, output_count, avg_latency_ms,
			anomaly_count, cost_per_accepted_output_with_review, summary_json
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, summary_id) DO UPDATE SET
			generated_at = excluded.generated_at,
			total_cost_usd = excluded.total_cost_usd,
			total_events = excluded.total_events,
			output_count = excluded.output_count,
			avg_latency_ms = excluded.avg_latency_ms,
			anomaly_count = excluded.anomaly_count,
			cost_per_accepted_output_with_review = excluded.cost_per_accepted_output_with_review,
			summary_json = excluded.summary_json
	`, summary.TenantID, summary.SummaryID, timePtrString(summary.PeriodStart),
		timePtrString(summary.PeriodEnd), formatTime(summary.GeneratedAt), summary.TotalCostUSD,
		summary.TotalEvents, summary.OutputCount, avg, summary.AnomalyCount,
		costPerAccepted, string(body))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) ListTokenEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, tokenEventSelect+`
		WHERE tenant_id = ?
		ORDER BY occurred_at DESC, event_id DESC
		LIMIT ?
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	return scanTokenEvents(rows)
}

func (r *SQLiteRepository) ListTokenEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, tokenEventSelect+`
		WHERE tenant_id = ? AND occurred_at < ?
		ORDER BY occurred_at DESC, event_id DESC
		LIMIT ?
	`, tenantID, formatTime(before), limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	return scanTokenEvents(rows)
}

func (r *SQLiteRepository) ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, anomaly_id, event_id, worker_id, detected_at, severity,
			type, description, observed_value, threshold_value, details_json
		FROM anomaly_signals
		WHERE tenant_id = ?
		ORDER BY detected_at DESC, anomaly_id DESC
		LIMIT ?
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()

	var signals []domain.AnomalySignal
	for rows.Next() {
		var signal domain.AnomalySignal
		var eventID, workerID, details sql.NullString
		var observed, threshold sql.NullFloat64
		var detected string
		if err := rows.Scan(&signal.TenantID, &signal.AnomalyID, &eventID, &workerID,
			&detected, &signal.Severity, &signal.Type, &signal.Description, &observed,
			&threshold, &details); err != nil {
			return nil, wrapDBErr(err)
		}
		signal.EventID = eventID.String
		signal.WorkerID = workerID.String
		signal.DetectedAt = parseTime(detected)
		if observed.Valid {
			signal.ObservedValue = &observed.Float64
		}
		if threshold.Valid {
			signal.ThresholdValue = &threshold.Float64
		}
		if details.Valid && details.String != "" {
			_ = json.Unmarshal([]byte(details.String), &signal.Details)
		}
		signals = append(signals, signal)
	}
	return signals, wrapDBErr(rows.Err())
}

func (r *SQLiteRepository) DeleteTenantData(ctx context.Context, tenantID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return wrapDBErr(err)
	}
	defer rollback(tx)
	for _, statement := range []string{
		`DELETE FROM productivity_summaries WHERE tenant_id = ?`,
		`DELETE FROM anomaly_signals WHERE tenant_id = ?`,
		`DELETE FROM cost_snapshots WHERE tenant_id = ?`,
		`DELETE FROM token_usage_events WHERE tenant_id = ?`,
		`DELETE FROM jobs WHERE tenant_id = ?`,
		`DELETE FROM workers WHERE tenant_id = ?`,
		`DELETE FROM tenants WHERE tenant_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, statement, tenantID); err != nil {
			return wrapDBErr(err)
		}
	}
	return wrapDBErr(tx.Commit())
}

const tokenEventSelect = `
	SELECT tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
		provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
		input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
		cost_is_degraded, cost_degraded_code, external_estimate_usd,
		external_estimate_currency, latency_ms, task_category, output_status,
		review_score, occurred_at, created_at, tags_json, idempotency_key
	FROM token_usage_events
`

func scanTokenEvents(rows *sql.Rows) ([]domain.TokenEvent, error) {
	var events []domain.TokenEvent
	for rows.Next() {
		var event domain.TokenEvent
		var jobID, sessionID, runID, costCode, externalCurrency, tags, idempotencyKey sql.NullString
		var cost, externalCost, reviewScore sql.NullFloat64
		var costIsDegraded int
		var occurredAt, createdAt string
		if err := rows.Scan(&event.TenantID, &event.EventID, &event.WorkerID, &event.WorkerName,
			&jobID, &sessionID, &runID, &event.Provider, &event.ModelID, &event.PromptTokens,
			&event.CompletionTokens, &event.CachedTokens, &event.InputTokens, &event.OutputTokens,
			&event.TotalTokens, &cost, &event.CostCurrency, &costIsDegraded, &costCode,
			&externalCost, &externalCurrency, &event.LatencyMs, &event.TaskCategory,
			&event.OutputStatus, &reviewScore, &occurredAt, &createdAt, &tags, &idempotencyKey); err != nil {
			return nil, wrapDBErr(err)
		}
		event.JobID = jobID.String
		event.SessionID = sessionID.String
		event.RunID = runID.String
		event.CostIsDegraded = costIsDegraded == 1
		event.CostDegradedCode = costCode.String
		event.Timestamp = parseTime(occurredAt)
		event.CreatedAt = parseTime(createdAt)
		event.TaskType = event.TaskCategory
		if cost.Valid {
			event.CostEstimateUSD = &cost.Float64
		}
		if externalCost.Valid {
			event.ExternalEstimate = &domain.ExternalEstimate{
				CostUSD:  externalCost.Float64,
				Currency: externalCurrency.String,
			}
		}
		if reviewScore.Valid {
			event.ReviewScore = &reviewScore.Float64
		}
		if tags.Valid && tags.String != "" {
			_ = json.Unmarshal([]byte(tags.String), &event.Tags)
		}
		if idempotencyKey.Valid {
			event.IdempotencyKey = idempotencyKey.String
		}
		events = append(events, event)
	}
	return events, wrapDBErr(rows.Err())
}

func marshalNullable(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(body) == "null" {
		return nil, nil
	}
	return string(body), nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func timePtrString(value *time.Time) interface{} {
	if value == nil || value.IsZero() {
		return nil
	}
	return formatTime(*value)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

func wrapDBErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrUnavailable, err)
}
