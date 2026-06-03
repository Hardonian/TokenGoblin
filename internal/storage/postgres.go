package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 30 * time.Minute
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("%w: connect to postgres: %v", ErrUnavailable, err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("%w: ping postgres: %v", ErrUnavailable, err)
	}
	if err := verifyPostgresRLS(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresRepository{pool: pool}, nil
}

func verifyPostgresRLS(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		"tenants",
		"workers",
		"jobs",
		"token_usage_events",
		"cost_snapshots",
		"anomaly_signals",
		"productivity_summaries",
		"tenant_pricing_overrides",
		"output_analyses",
		"tenant_members",
		"audit_events",
		"recommendation_states",
		"api_keys",
	}
	var missing []string
	for _, table := range tables {
		var enabled bool
		if err := pool.QueryRow(ctx, `SELECT COALESCE((SELECT relrowsecurity FROM pg_class WHERE oid = to_regclass($1)), false)`, table).Scan(&enabled); err != nil {
			return fmt.Errorf("%w: verify postgres rls for %s: %v", ErrUnavailable, table, err)
		}
		if !enabled {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: postgres row level security is not enabled for: %s", ErrUnavailable, strings.Join(missing, ", "))
	}
	return nil
}

func (r *PostgresRepository) Close() error {
	r.pool.Close()
	return nil
}

func (r *PostgresRepository) UpsertTenant(ctx context.Context, tenant domain.Tenant) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(tenant_id) DO UPDATE SET
			name = EXCLUDED.name,
			tier = EXCLUDED.tier,
			usage_limit_usd = EXCLUDED.usage_limit_usd,
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			updated_at = EXCLUDED.updated_at
	`, tenant.TenantID, tenant.Name, tenant.Tier, tenant.UsageLimitUSD, nullString(tenant.StripeCustomerID), nullString(tenant.StripeSubscriptionID), tenant.CreatedAt, tenant.UpdatedAt)
	return wrapDBErr(err)
}

func (r *PostgresRepository) GetTenant(ctx context.Context, tenantID string) (*domain.Tenant, error) {
	var t domain.Tenant
	var stripeCust, stripeSub *string
	err := r.pool.QueryRow(ctx, `
		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM tenants
		WHERE tenant_id = $1
	`, tenantID).Scan(&t.TenantID, &t.Name, &t.Tier, &t.UsageLimitUSD, &stripeCust, &stripeSub, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	if stripeCust != nil {
		t.StripeCustomerID = *stripeCust
	}
	if stripeSub != nil {
		t.StripeSubscriptionID = *stripeSub
	}
	return &t, nil
}

func (r *PostgresRepository) GetTenantCurrentMonthCost(ctx context.Context, tenantID string) (float64, error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var total *float64
	err := r.pool.QueryRow(ctx, `
		SELECT SUM(cost_estimate_usd)
		FROM token_usage_events
		WHERE tenant_id = $1 AND occurred_at >= $2
	`, tenantID, startOfMonth).Scan(&total)

	if err != nil {
		return 0, wrapDBErr(err)
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

func (r *PostgresRepository) GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error) {
	var point domain.PricePoint
	var created time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT provider, model_id, prompt_price_per_million, completion_price_per_million, created_at
		FROM tenant_pricing_overrides
		WHERE tenant_id = $1 AND provider = $2 AND model_id = $3
	`, tenantID, provider, modelID).Scan(&point.Provider, &point.ModelID, &point.InputCostPerMillion, &point.OutputCostPerMillion, &created)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	point.Currency = "USD"
	point.Source = "override"
	point.EffectiveFrom = created
	point.CachedInputCostPerMillion = point.InputCostPerMillion / 2.0 // Simple default logic for overrides
	return &point, nil
}

func (r *PostgresRepository) SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error {
	overrideID := tenantID + ":" + point.Provider + ":" + point.ModelID
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenant_pricing_overrides (override_id, tenant_id, provider, model_id, prompt_price_per_million, completion_price_per_million, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(tenant_id, provider, model_id) DO UPDATE SET
			prompt_price_per_million = EXCLUDED.prompt_price_per_million,
			completion_price_per_million = EXCLUDED.completion_price_per_million,
			created_at = EXCLUDED.created_at
	`, overrideID, tenantID, point.Provider, point.ModelID, point.InputCostPerMillion, point.OutputCostPerMillion, time.Now().UTC())
	return wrapDBErr(err)
}

func (r *PostgresRepository) ListPricingOverrides(ctx context.Context, tenantID string) ([]domain.PricePoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT provider, model_id, prompt_price_per_million, completion_price_per_million, created_at
		FROM tenant_pricing_overrides
		WHERE tenant_id = $1
<<<<<<< Updated upstream
		ORDER BY provider, model_id
=======
		ORDER BY created_at DESC
>>>>>>> Stashed changes
	`, tenantID)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	var points []domain.PricePoint
	for rows.Next() {
<<<<<<< Updated upstream
		var point domain.PricePoint
		var created time.Time
		if err := rows.Scan(&point.Provider, &point.ModelID, &point.InputCostPerMillion, &point.OutputCostPerMillion, &created); err != nil {
			return nil, wrapDBErr(err)
		}
		point.Currency = "USD"
		point.Source = "override"
		point.EffectiveFrom = created
		point.CachedInputCostPerMillion = point.InputCostPerMillion / 2.0
		points = append(points, point)
	}
	return points, wrapDBErr(rows.Err())
}
=======
		var p domain.PricePoint
		var created time.Time
		if err := rows.Scan(&p.Provider, &p.ModelID, &p.InputCostPerMillion, &p.OutputCostPerMillion, &created); err != nil {
			return nil, wrapDBErr(err)
		}
		p.EffectiveFrom = created
		points = append(points, p)
	}
	return points, wrapDBErr(rows.Err())
}



>>>>>>> Stashed changes

func (r *PostgresRepository) DeleteOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).UTC()
	res, err := r.pool.Exec(ctx, `DELETE FROM token_usage_events WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, wrapDBErr(err)
	}
	return res.RowsAffected(), nil
}

func (r *PostgresRepository) DeleteTenantData(ctx context.Context, tenantID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tenants WHERE tenant_id = $1`, tenantID)
	return wrapDBErr(err)
}

func (r *PostgresRepository) SaveAPIKey(ctx context.Context, key domain.APIKey) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO api_keys (key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, key.KeyID, key.TenantID, key.Name, key.KeyHash, normalizeRole(key.Role), key.CreatedAt, key.LastUsedAt, key.IsRevoked)
	return wrapDBErr(err)
}

func (r *PostgresRepository) GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.pool.QueryRow(ctx, `
		SELECT key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked
		FROM api_keys
		WHERE key_id = $1
	`, keyID).Scan(&key.KeyID, &key.TenantID, &key.Name, &key.KeyHash, &key.Role, &key.CreatedAt, &key.LastUsedAt, &key.IsRevoked)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	key.Role = normalizeRole(key.Role)
	return &key, nil
}

func (r *PostgresRepository) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE api_keys
		SET last_used_at = $1
		WHERE key_id = $2
	`, time.Now().UTC(), keyID)
	return wrapDBErr(err)
}

func (r *PostgresRepository) UpsertTenantMember(ctx context.Context, member domain.TenantMember) error {
	now := member.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	createdAt := member.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenant_members (tenant_id, subject_id, email, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(tenant_id, subject_id) DO UPDATE SET
			email = EXCLUDED.email,
			role = EXCLUDED.role,
			updated_at = EXCLUDED.updated_at
	`, member.TenantID, member.SubjectID, nullString(member.Email), normalizeRole(member.Role), createdAt, now)
	return wrapDBErr(err)
}

func (r *PostgresRepository) ListTenantMembers(ctx context.Context, tenantID string) ([]domain.TenantMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tenant_id, subject_id, email, role, created_at, updated_at
		FROM tenant_members
		WHERE tenant_id = $1
		ORDER BY role, subject_id
	`, tenantID)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	var members []domain.TenantMember
	for rows.Next() {
		var member domain.TenantMember
		var email *string
		if err := rows.Scan(&member.TenantID, &member.SubjectID, &email, &member.Role, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return nil, wrapDBErr(err)
		}
		if email != nil {
			member.Email = *email
		}
		member.Role = normalizeRole(member.Role)
		members = append(members, member)
	}
	return members, wrapDBErr(rows.Err())
}

func (r *PostgresRepository) SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	metadataJSON, err := marshalNullable(event.Metadata)
	if err != nil {
		return err
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO audit_events (tenant_id, event_id, type, actor, resource, metadata_json, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.TenantID, event.EventID, event.Type, event.Actor, nullString(event.Resource), metadataJSON, event.Timestamp)
	return wrapDBErr(err)
}

func (r *PostgresRepository) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT tenant_id, event_id, type, actor, resource, metadata_json, created_at
		FROM audit_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC, event_id DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	var events []domain.AuditEvent
	for rows.Next() {
		var event domain.AuditEvent
		var resource *string
		var metadataJSON []byte
		if err := rows.Scan(&event.TenantID, &event.EventID, &event.Type, &event.Actor, &resource, &metadataJSON, &event.Timestamp); err != nil {
			return nil, wrapDBErr(err)
		}
		if resource != nil {
			event.Resource = *resource
		}
		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &event.Metadata)
		}
		events = append(events, event)
	}
	return events, wrapDBErr(rows.Err())
}

func (r *PostgresRepository) SetRecommendationState(ctx context.Context, state domain.RecommendationState) error {
	now := state.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	createdAt := state.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO recommendation_states (tenant_id, recommendation_id, status, actor, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(tenant_id, recommendation_id) DO UPDATE SET
			status = EXCLUDED.status,
			actor = EXCLUDED.actor,
			note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at
	`, state.TenantID, state.RecommendationID, state.Status, nullString(state.Actor), nullString(state.Note), createdAt, now)
	return wrapDBErr(err)
}

func (r *PostgresRepository) ListRecommendationStates(ctx context.Context, tenantID string) ([]domain.RecommendationState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tenant_id, recommendation_id, status, actor, note, created_at, updated_at
		FROM recommendation_states
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	var states []domain.RecommendationState
	for rows.Next() {
		var state domain.RecommendationState
		var actor, note *string
		if err := rows.Scan(&state.TenantID, &state.RecommendationID, &state.Status, &actor, &note, &state.CreatedAt, &state.UpdatedAt); err != nil {
			return nil, wrapDBErr(err)
		}
		if actor != nil {
			state.Actor = *actor
		}
		if note != nil {
			state.Note = *note
		}
		states = append(states, state)
	}
	return states, wrapDBErr(rows.Err())
}

func (r *PostgresRepository) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrapDBErr(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	var tagsJSON interface{}
	if event.TagsJSON != nil {
		tagsJSON = string(event.TagsJSON)
	} else {
		tagsJSON, err = marshalNullable(event.GetTags())
	}
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
			review_score, occurred_at, created_at, prompt_excerpt, output_excerpt,
			prompt_reference, output_reference, tags_json, idempotency_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33)
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
			prompt_excerpt = EXCLUDED.prompt_excerpt,
			output_excerpt = EXCLUDED.output_excerpt,
			prompt_reference = EXCLUDED.prompt_reference,
			output_reference = EXCLUDED.output_reference,
			tags_json = EXCLUDED.tags_json,
			idempotency_key = EXCLUDED.idempotency_key
	`, event.TenantID, event.EventID, event.WorkerID, workerName, nullString(event.JobID),
		nullString(event.SessionID), nullString(event.RunID), event.Provider, event.ModelID,
		event.PromptTokens, event.CompletionTokens, event.CachedTokens, event.InputTokens,
		event.OutputTokens, event.TotalTokens, event.CostEstimateUSD, event.CostCurrency,
		event.CostIsDegraded, nullString(event.CostDegradedCode), externalCost,
		externalCurrency, event.LatencyMs, taskCategory, string(event.OutputStatus),
		event.ReviewScore, event.Timestamp, now, nullString(event.PromptExcerpt),
		nullString(event.OutputExcerpt), nullString(event.PromptReference), nullString(event.OutputReference),
		tagsJSON, nullString(event.IdempotencyKey))
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

func (r *PostgresRepository) SaveOutputAnalysis(ctx context.Context, analysis domain.OutputAnalysis) error {
	issuesJSON, err := json.Marshal(analysis.Issues)
	if err != nil {
		return err
	}
	recsJSON, err := json.Marshal(analysis.Recommendations)
	if err != nil {
		return err
	}
	evidenceJSON, err := json.Marshal(analysis.Evidence)
	if err != nil {
		return err
	}
	degradedJSON, err := marshalNullable(analysis.Degraded)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO output_analyses (
			tenant_id, analysis_id, event_id, worker_id, analyzed_at,
			efficiency_score, goblin_score, issues_json, recommendations_json,
			evidence_json, degraded_json, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT(tenant_id, analysis_id) DO UPDATE SET
			analyzed_at = EXCLUDED.analyzed_at,
			efficiency_score = EXCLUDED.efficiency_score,
			goblin_score = EXCLUDED.goblin_score,
			issues_json = EXCLUDED.issues_json,
			recommendations_json = EXCLUDED.recommendations_json,
			evidence_json = EXCLUDED.evidence_json,
			degraded_json = EXCLUDED.degraded_json
	`, analysis.TenantID, analysis.AnalysisID, analysis.EventID, analysis.WorkerID,
		analysis.AnalyzedAt, analysis.EfficiencyScore, analysis.GoblinScore,
		string(issuesJSON), string(recsJSON), string(evidenceJSON), degradedJSON, time.Now().UTC())
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

func (r *PostgresRepository) ListOutputAnalyses(ctx context.Context, tenantID string, limit int) ([]domain.OutputAnalysis, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, outputAnalysisSelectPostgres+`
		WHERE tenant_id = $1
		ORDER BY analyzed_at DESC, analysis_id DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	return scanOutputAnalysesPostgres(rows)
}

func (r *PostgresRepository) ListOutputAnalysesByWorker(ctx context.Context, tenantID, workerID string, limit int) ([]domain.OutputAnalysis, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, outputAnalysisSelectPostgres+`
		WHERE tenant_id = $1 AND worker_id = $2
		ORDER BY analyzed_at DESC, analysis_id DESC
		LIMIT $3
	`, tenantID, workerID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	return scanOutputAnalysesPostgres(rows)
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
		if details != nil && *details != "" && *details != "{}" {
			_ = json.Unmarshal([]byte(*details), &signal.Details)
		}
		signals = append(signals, signal)
	}
	return signals, wrapDBErr(rows.Err())
}

const outputAnalysisSelectPostgres = `
	SELECT tenant_id, analysis_id, event_id, worker_id, analyzed_at,
		efficiency_score, goblin_score, issues_json, recommendations_json,
		evidence_json, degraded_json
	FROM output_analyses
`

func scanOutputAnalysesPostgres(rows pgx.Rows) ([]domain.OutputAnalysis, error) {
	var analyses []domain.OutputAnalysis
	for rows.Next() {
		var analysis domain.OutputAnalysis
		var issuesJSON, recommendationsJSON, evidenceJSON string
		var degradedJSON *string
		if err := rows.Scan(
			&analysis.TenantID,
			&analysis.AnalysisID,
			&analysis.EventID,
			&analysis.WorkerID,
			&analysis.AnalyzedAt,
			&analysis.EfficiencyScore,
			&analysis.GoblinScore,
			&issuesJSON,
			&recommendationsJSON,
			&evidenceJSON,
			&degradedJSON,
		); err != nil {
			return nil, wrapDBErr(err)
		}
		if issuesJSON != "" {
			if err := json.Unmarshal([]byte(issuesJSON), &analysis.Issues); err != nil {
				return nil, wrapDBErr(err)
			}
		}
		if recommendationsJSON != "" {
			if err := json.Unmarshal([]byte(recommendationsJSON), &analysis.Recommendations); err != nil {
				return nil, wrapDBErr(err)
			}
		}
		if evidenceJSON != "" {
			if err := json.Unmarshal([]byte(evidenceJSON), &analysis.Evidence); err != nil {
				return nil, wrapDBErr(err)
			}
		}
		if degradedJSON != nil && *degradedJSON != "" && *degradedJSON != "[]" && *degradedJSON != "null" {
			if err := json.Unmarshal([]byte(*degradedJSON), &analysis.Degraded); err != nil {
				return nil, wrapDBErr(err)
			}
		}
		analyses = append(analyses, analysis)
	}
	return analyses, wrapDBErr(rows.Err())
}

const tokenEventSelectPostgres = `
	SELECT tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
		provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
		input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
		cost_is_degraded, cost_degraded_code, external_estimate_usd,
		external_estimate_currency, latency_ms, task_category, output_status,
		review_score, occurred_at, created_at, prompt_excerpt, output_excerpt,
		prompt_reference, output_reference, tags_json, idempotency_key
	FROM token_usage_events
`

const outputAnalysisSelectPostgres = `
	SELECT tenant_id, analysis_id, event_id, worker_id, analyzed_at,
		efficiency_score, goblin_score, issues_json, recommendations_json,
		evidence_json, degraded_json, created_at
	FROM output_analyses
`

func scanTokenEventsPostgres(rows pgx.Rows) ([]domain.TokenEvent, error) {
	var events []domain.TokenEvent
	for rows.Next() {
		var event domain.TokenEvent
		var jobID, sessionID, runID, costCode, externalCurrency, promptExcerpt, outputExcerpt, promptReference, outputReference, tags, idempotencyKey *string
		var externalCost *float64
		if err := rows.Scan(&event.TenantID, &event.EventID, &event.WorkerID, &event.WorkerName,
			&jobID, &sessionID, &runID, &event.Provider, &event.ModelID, &event.PromptTokens,
			&event.CompletionTokens, &event.CachedTokens, &event.InputTokens, &event.OutputTokens,
			&event.TotalTokens, &event.CostEstimateUSD, &event.CostCurrency, &event.CostIsDegraded, &costCode,
			&externalCost, &externalCurrency, &event.LatencyMs, &event.TaskCategory,
			&event.OutputStatus, &event.ReviewScore, &event.Timestamp, &event.CreatedAt, &promptExcerpt,
			&outputExcerpt, &promptReference, &outputReference, &tags, &idempotencyKey); err != nil {
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
		if promptExcerpt != nil {
			event.PromptExcerpt = *promptExcerpt
		}
		if outputExcerpt != nil {
			event.OutputExcerpt = *outputExcerpt
		}
		if promptReference != nil {
			event.PromptReference = *promptReference
		}
		if outputReference != nil {
			event.OutputReference = *outputReference
		}
		event.TaskType = event.TaskCategory
		if externalCost != nil {
			event.ExternalEstimate = &domain.ExternalEstimate{
				CostUSD: *externalCost,
			}
			if externalCurrency != nil {
				event.ExternalEstimate.Currency = *externalCurrency
			}
		}
		if tags != nil && *tags != "" && *tags != "{}" {
			_ = json.Unmarshal([]byte(*tags), &event.Tags)
		}
		if idempotencyKey != nil {
			event.IdempotencyKey = *idempotencyKey
		}
		events = append(events, event)
	}
	return events, wrapDBErr(rows.Err())
}

func (r *PostgresRepository) SaveRecommendationDecision(ctx context.Context, tenantID, recID, status string) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO recommendation_states (tenant_id, recommendation_id, status, actor, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(tenant_id, recommendation_id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, tenantID, recID, status, nil, nil, now, now)
	return wrapDBErr(err)
}

func (r *PostgresRepository) GetRecommendationDecisions(ctx context.Context, tenantID string) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT recommendation_id, status
		FROM recommendation_states
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()
	decisions := make(map[string]string)
	for rows.Next() {
		var recID, status string
		if err := rows.Scan(&recID, &status); err != nil {
			return nil, wrapDBErr(err)
		}
		decisions[recID] = status
	}
	return decisions, wrapDBErr(rows.Err())
}

func (r *PostgresRepository) GetTenantByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Tenant, error) {
	var tenant domain.Tenant
	var stripeCustID, stripeSubID *string
	err := r.pool.QueryRow(ctx, `
		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM tenants
		WHERE stripe_customer_id = $1
	`, stripeCustomerID).Scan(&tenant.TenantID, &tenant.Name, &tenant.Tier, &tenant.UsageLimitUSD, &stripeCustID, &stripeSubID, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	if stripeCustID != nil {
		tenant.StripeCustomerID = *stripeCustID
	}
	if stripeSubID != nil {
		tenant.StripeSubscriptionID = *stripeSubID
	}
	return &tenant, nil
}

func (r *PostgresRepository) GetTenantByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (*domain.Tenant, error) {
	var tenant domain.Tenant
	var stripeCustID, stripeSubID *string
	err := r.pool.QueryRow(ctx, `
		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM tenants
		WHERE stripe_subscription_id = $1
	`, stripeSubscriptionID).Scan(&tenant.TenantID, &tenant.Name, &tenant.Tier, &tenant.UsageLimitUSD, &stripeCustID, &stripeSubID, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	if stripeCustID != nil {
		tenant.StripeCustomerID = *stripeCustID
	}
	if stripeSubID != nil {
		tenant.StripeSubscriptionID = *stripeSubID
	}
	return &tenant, nil
}

func scanOutputAnalysesPostgres(rows pgx.Rows) ([]domain.OutputAnalysis, error) {
	var analyses []domain.OutputAnalysis
	for rows.Next() {
		var a domain.OutputAnalysis
		var issuesJSON, recsJSON, evJSON, degJSON string
		var analyzedAt, createdAt string
		if err := rows.Scan(&a.TenantID, &a.AnalysisID, &a.EventID, &a.WorkerID, &analyzedAt,
			&a.EfficiencyScore, &a.GoblinScore, &issuesJSON, &recsJSON, &evJSON, &degJSON, &createdAt); err != nil {
			return nil, wrapDBErr(err)
		}
		a.AnalyzedAt = parseTime(analyzedAt)
		_ = json.Unmarshal([]byte(issuesJSON), &a.Issues)
		_ = json.Unmarshal([]byte(recsJSON), &a.Recommendations)
		_ = json.Unmarshal([]byte(evJSON), &a.Evidence)
		if degJSON != "" {
			_ = json.Unmarshal([]byte(degJSON), &a.Degraded)
		}
		analyses = append(analyses, a)
	}
	return analyses, wrapDBErr(rows.Err())
}

