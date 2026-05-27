package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func OpenPostgres(ctx context.Context, dsn string) (*PostgresRepository, error) {
	if dsn == "" {
		return nil, fmt.Errorf("%w: missing database DSN", ErrUnavailable)
	}

	m, err := migrate.New(
		"file://data/migrations",
		dsn,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: create migrator: %v", ErrUnavailable, err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("%w: migrate up: %v", ErrUnavailable, err)
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: parse dsn: %v", ErrUnavailable, err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("%w: connect to postgres: %v", ErrUnavailable, err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("%w: ping postgres: %v", ErrUnavailable, err)
	}

	return &PostgresRepository{pool: pool}, nil
}

func (r *PostgresRepository) Close() error {
	r.pool.Close()
	return nil
}

func (r *PostgresRepository) UpsertTenant(ctx context.Context, tenant domain.Tenant) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(tenant_id) DO UPDATE SET
			name = EXCLUDED.name,
			updated_at = EXCLUDED.updated_at
	`, tenant.TenantID, tenant.Name, tenant.CreatedAt, tenant.UpdatedAt)
	return wrapDBErr(err)
}

func (r *PostgresRepository) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrapDBErr(err)
	}
	defer tx.Rollback(ctx)

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
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenants (tenant_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(tenant_id) DO UPDATE SET updated_at = EXCLUDED.updated_at
	`, tenant.TenantID, tenant.Name, tenant.CreatedAt, tenant.UpdatedAt); err != nil {
		return wrapDBErr(err)
	}

	workerName := event.WorkerName
	if workerName == "" {
		workerName = event.WorkerID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO workers (tenant_id, worker_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(tenant_id, worker_id) DO UPDATE SET
			name = EXCLUDED.name,
			updated_at = EXCLUDED.updated_at
	`, event.TenantID, event.WorkerID, workerName, now, now); err != nil {
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
		if _, err := tx.Exec(ctx, `
			INSERT INTO jobs (tenant_id, job_id, worker_id, name, task_category, status, started_at, ended_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT(tenant_id, job_id) DO UPDATE SET
				worker_id = EXCLUDED.worker_id,
				task_category = EXCLUDED.task_category,
				updated_at = EXCLUDED.updated_at
		`, event.TenantID, event.JobID, event.WorkerID, event.JobID, taskCategory, "active", nil, nil, now, now); err != nil {
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

	_, err = tx.Exec(ctx, `
		INSERT INTO token_usage_events (
			tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
			provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
			input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
			cost_is_degraded, cost_degraded_code, external_estimate_usd,
			external_estimate_currency, latency_ms, task_category, output_status,
			review_score, occurred_at, created_at, tags_json
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
		ON CONFLICT(tenant_id, event_id) DO UPDATE SET
			worker_id = EXCLUDED.worker_id,
			worker_name = EXCLUDED.worker_name,
			job_id = EXCLUDED.job_id,
			session_id = EXCLUDED.session_id,
			run_id = EXCLUDED.run_id,
			provider = EXCLUDED.provider,
			model_id = EXCLUDED.model_id,
			prompt_tokens = EXCLUDED.prompt_tokens,
			completion_tokens = EXCLUDED.completion_tokens,
			cached_tokens = EXCLUDED.cached_tokens,
			input_tokens = EXCLUDED.input_tokens,
			output_tokens = EXCLUDED.output_tokens,
			total_tokens = EXCLUDED.total_tokens,
			cost_estimate_usd = EXCLUDED.cost_estimate_usd,
			cost_currency = EXCLUDED.cost_currency,
			cost_is_degraded = EXCLUDED.cost_is_degraded,
			cost_degraded_code = EXCLUDED.cost_degraded_code,
			external_estimate_usd = EXCLUDED.external_estimate_usd,
			external_estimate_currency = EXCLUDED.external_estimate_currency,
			latency_ms = EXCLUDED.latency_ms,
			task_category = EXCLUDED.task_category,
			output_status = EXCLUDED.output_status,
			review_score = EXCLUDED.review_score,
			occurred_at = EXCLUDED.occurred_at,
			tags_json = EXCLUDED.tags_json
	`, event.TenantID, event.EventID, event.WorkerID, workerName, nullString(event.JobID),
		nullString(event.SessionID), nullString(event.RunID), event.Provider, event.ModelID,
		event.PromptTokens, event.CompletionTokens, event.CachedTokens, event.InputTokens,
		event.OutputTokens, event.TotalTokens, event.CostEstimateUSD, event.CostCurrency,
		event.CostIsDegraded, nullString(event.CostDegradedCode), externalCost,
		externalCurrency, event.LatencyMs, taskCategory, string(event.OutputStatus),
		event.ReviewScore, event.Timestamp, now, tagsJSON)
	if err != nil {
		return wrapDBErr(err)
	}

	return wrapDBErr(tx.Commit(ctx))
}

func (r *PostgresRepository) SaveCostSnapshot(ctx context.Context, snapshot domain.CostSnapshot) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO cost_snapshots (
			tenant_id, snapshot_id, event_id, provider, model_id, input_tokens,
			output_tokens, cached_tokens, cost_estimate_usd, currency, is_degraded,
			degraded_code, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT(tenant_id, snapshot_id) DO UPDATE SET
			cost_estimate_usd = EXCLUDED.cost_estimate_usd,
			is_degraded = EXCLUDED.is_degraded,
			degraded_code = EXCLUDED.degraded_code,
			created_at = EXCLUDED.created_at
	`, snapshot.TenantID, snapshot.SnapshotID, snapshot.EventID, snapshot.Provider,
		snapshot.ModelID, snapshot.InputTokens, snapshot.OutputTokens, snapshot.CachedTokens,
		snapshot.CostEstimateUSD, snapshot.Currency, snapshot.IsDegraded,
		nullString(snapshot.DegradedCode), snapshot.CreatedAt)
	return wrapDBErr(err)
}

func (r *PostgresRepository) SaveAnomalySignal(ctx context.Context, signal domain.AnomalySignal) error {
	detailsJSON, err := marshalNullable(signal.Details)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO anomaly_signals (
			tenant_id, anomaly_id, event_id, worker_id, detected_at, severity,
			type, description, observed_value, threshold_value, details_json, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT(tenant_id, anomaly_id) DO UPDATE SET
			detected_at = EXCLUDED.detected_at,
			severity = EXCLUDED.severity,
			description = EXCLUDED.description,
			observed_value = EXCLUDED.observed_value,
			threshold_value = EXCLUDED.threshold_value,
			details_json = EXCLUDED.details_json,
			created_at = EXCLUDED.created_at
	`, signal.TenantID, signal.AnomalyID, nullString(signal.EventID), nullString(signal.WorkerID),
		signal.DetectedAt, string(signal.Severity), string(signal.Type),
		signal.Description, signal.ObservedValue, signal.ThresholdValue, detailsJSON,
		signal.DetectedAt)
	return wrapDBErr(err)
}

func (r *PostgresRepository) SaveProductivitySummary(ctx context.Context, summary domain.ProductivitySummary) error {
	body, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO productivity_summaries (
			tenant_id, summary_id, period_start, period_end, generated_at,
			total_cost_usd, total_events, output_count, avg_latency_ms,
			anomaly_count, cost_per_accepted_output_with_review, summary_json
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT(tenant_id, summary_id) DO UPDATE SET
			generated_at = EXCLUDED.generated_at,
			total_cost_usd = EXCLUDED.total_cost_usd,
			total_events = EXCLUDED.total_events,
			output_count = EXCLUDED.output_count,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			anomaly_count = EXCLUDED.anomaly_count,
			cost_per_accepted_output_with_review = EXCLUDED.cost_per_accepted_output_with_review,
			summary_json = EXCLUDED.summary_json
	`, summary.TenantID, summary.SummaryID, summary.PeriodStart,
		summary.PeriodEnd, summary.GeneratedAt, summary.TotalCostUSD,
		summary.TotalEvents, summary.OutputCount, summary.AvgLatencyMs, summary.AnomalyCount,
		summary.CostPerAcceptedOutputWithReview, string(body))
	return wrapDBErr(err)
}

func (r *PostgresRepository) ListTokenEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, tokenEventSelectPostgres+`
		WHERE tenant_id = $1
		ORDER BY occurred_at DESC, event_id DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	return scanTokenEventsPostgres(rows)
}

func (r *PostgresRepository) ListTokenEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, tokenEventSelectPostgres+`
		WHERE tenant_id = $1 AND occurred_at < $2
		ORDER BY occurred_at DESC, event_id DESC
		LIMIT $3
	`, tenantID, before, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	return scanTokenEventsPostgres(rows)
}

func (r *PostgresRepository) ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT tenant_id, anomaly_id, event_id, worker_id, detected_at, severity,
			type, description, observed_value, threshold_value, details_json
		FROM anomaly_signals
		WHERE tenant_id = $1
		ORDER BY detected_at DESC, anomaly_id DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()

	var signals []domain.AnomalySignal
	for rows.Next() {
		var signal domain.AnomalySignal
		var eventID, workerID, details *string
		if err := rows.Scan(&signal.TenantID, &signal.AnomalyID, &eventID, &workerID,
			&signal.DetectedAt, &signal.Severity, &signal.Type, &signal.Description, &signal.ObservedValue,
			&signal.ThresholdValue, &details); err != nil {
			return nil, wrapDBErr(err)
		}
		if eventID != nil {
			signal.EventID = *eventID
		}
		if workerID != nil {
			signal.WorkerID = *workerID
		}
		if details != nil && *details != "" {
			_ = json.Unmarshal([]byte(*details), &signal.Details)
		}
		signals = append(signals, signal)
	}
	return signals, wrapDBErr(rows.Err())
}

func (r *PostgresRepository) DeleteTenantData(ctx context.Context, tenantID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrapDBErr(err)
	}
	defer tx.Rollback(ctx)
	for _, statement := range []string{
		`DELETE FROM productivity_summaries WHERE tenant_id = $1`,
		`DELETE FROM anomaly_signals WHERE tenant_id = $1`,
		`DELETE FROM cost_snapshots WHERE tenant_id = $1`,
		`DELETE FROM token_usage_events WHERE tenant_id = $1`,
		`DELETE FROM jobs WHERE tenant_id = $1`,
		`DELETE FROM workers WHERE tenant_id = $1`,
		`DELETE FROM tenants WHERE tenant_id = $1`,
	} {
		if _, err := tx.Exec(ctx, statement, tenantID); err != nil {
			return wrapDBErr(err)
		}
	}
	return wrapDBErr(tx.Commit(ctx))
}

const tokenEventSelectPostgres = `
	SELECT tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
		provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
		input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
		cost_is_degraded, cost_degraded_code, external_estimate_usd,
		external_estimate_currency, latency_ms, task_category, output_status,
		review_score, occurred_at, created_at, tags_json
	FROM token_usage_events
`

func scanTokenEventsPostgres(rows pgx.Rows) ([]domain.TokenEvent, error) {
	var events []domain.TokenEvent
	for rows.Next() {
		var event domain.TokenEvent
		var jobID, sessionID, runID, costCode, externalCurrency, tags *string
		var externalCost *float64
		if err := rows.Scan(&event.TenantID, &event.EventID, &event.WorkerID, &event.WorkerName,
			&jobID, &sessionID, &runID, &event.Provider, &event.ModelID, &event.PromptTokens,
			&event.CompletionTokens, &event.CachedTokens, &event.InputTokens, &event.OutputTokens,
			&event.TotalTokens, &event.CostEstimateUSD, &event.CostCurrency, &event.CostIsDegraded, &costCode,
			&externalCost, &externalCurrency, &event.LatencyMs, &event.TaskCategory,
			&event.OutputStatus, &event.ReviewScore, &event.Timestamp, &event.CreatedAt, &tags); err != nil {
			return nil, wrapDBErr(err)
		}
		if jobID != nil {
			event.JobID = *jobID
		}
		if sessionID != nil {
			event.SessionID = *sessionID
		}
		if runID != nil {
			event.RunID = *runID
		}
		if costCode != nil {
			event.CostDegradedCode = *costCode
		}
		event.TaskType = event.TaskCategory
		if externalCost != nil {
			event.ExternalEstimate = &domain.ExternalEstimate{
				CostUSD:  *externalCost,
			}
			if externalCurrency != nil {
				event.ExternalEstimate.Currency = *externalCurrency
			}
		}
		if tags != nil && *tags != "" {
			_ = json.Unmarshal([]byte(*tags), &event.Tags)
		}
		events = append(events, event)
	}
	return events, wrapDBErr(rows.Err())
}
