package storage

import (
	"context"
	"strings"
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

	// Append pragmas for WAL mode, busy timeout, and synchronous mode
	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: open sqlite: %v", ErrUnavailable, err)
	}
	repo := &SQLiteRepository{db: db}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := repo.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *SQLiteRepository) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS tenants (
			tenant_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			tier TEXT NOT NULL DEFAULT 'free',
			usage_limit_usd REAL NOT NULL DEFAULT 10.00,
			stripe_customer_id TEXT,
			stripe_subscription_id TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tenant_pricing_overrides (
			override_id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			provider TEXT NOT NULL,
			model_id TEXT NOT NULL,
			prompt_price_per_million REAL NOT NULL,
			completion_price_per_million REAL NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(tenant_id, provider, model_id)
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
			prompt_excerpt TEXT,
			output_excerpt TEXT,
			prompt_reference TEXT,
			output_reference TEXT,
			tags_json TEXT,
			idempotency_key TEXT,
			fingerprint TEXT,
			PRIMARY KEY (tenant_id, event_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			FOREIGN KEY (tenant_id, worker_id) REFERENCES workers(tenant_id, worker_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_token_usage_tenant_model ON token_usage_events (tenant_id, model_id);`,

		`CREATE INDEX IF NOT EXISTS idx_token_usage_tenant_fingerprint ON token_usage_events (tenant_id, fingerprint);`,

		`CREATE INDEX IF NOT EXISTS idx_jobs_tenant_status ON jobs (tenant_id, status);`,

		`CREATE TABLE IF NOT EXISTS api_keys (
			key_id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT,
			is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS tenant_members (
			tenant_id TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			email TEXT,
			role TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, subject_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_role ON tenant_members(tenant_id, role);`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			tenant_id TEXT NOT NULL,
			event_id TEXT NOT NULL,
			type TEXT NOT NULL,
			actor TEXT NOT NULL,
			resource TEXT,
			metadata_json TEXT,
			created_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, event_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created ON audit_events(tenant_id, created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS recommendation_states (
			tenant_id TEXT NOT NULL,
			recommendation_id TEXT NOT NULL,
			status TEXT NOT NULL,
			actor TEXT,
			note TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, recommendation_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
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
		`CREATE TABLE IF NOT EXISTS output_analyses (
			tenant_id TEXT NOT NULL,
			analysis_id TEXT NOT NULL,
			event_id TEXT NOT NULL,
			worker_id TEXT NOT NULL,
			analyzed_at TEXT NOT NULL,
			efficiency_score INTEGER NOT NULL,
			goblin_score INTEGER NOT NULL,
			issues_json TEXT NOT NULL,
			recommendations_json TEXT NOT NULL,
			evidence_json TEXT NOT NULL,
			degraded_json TEXT,
			created_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, analysis_id),
			FOREIGN KEY (tenant_id, event_id) REFERENCES token_usage_events(tenant_id, event_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_output_analyses_tenant_analyzed ON output_analyses (tenant_id, analyzed_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_output_analyses_tenant_worker ON output_analyses (tenant_id, worker_id, analyzed_at DESC);`,
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
		`CREATE TABLE IF NOT EXISTS agents (
			tenant_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			owner_id TEXT,
			agent_type TEXT NOT NULL,
			framework TEXT,
			status TEXT NOT NULL,
			budget_usd REAL,
			budget_period TEXT,
			sla_latency_ms INTEGER,
			sla_success_rate REAL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			retired_at TEXT,
			retirement_reason TEXT,
			PRIMARY KEY (tenant_id, agent_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS agent_performance_reviews (
			tenant_id TEXT NOT NULL,
			review_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			period_start TEXT NOT NULL,
			period_end TEXT NOT NULL,
			event_count INTEGER NOT NULL,
			total_cost_usd REAL NOT NULL,
			acceptance_rate REAL NOT NULL,
			avg_latency_ms REAL NOT NULL,
			cost_per_outcome REAL NOT NULL,
			sla_violations INTEGER NOT NULL,
			efficiency_grade TEXT NOT NULL,
			recommendation TEXT NOT NULL,
			generated_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, review_id),
			FOREIGN KEY (tenant_id, agent_id) REFERENCES agents(tenant_id, agent_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS governance_policies (
			tenant_id TEXT NOT NULL,
			policy_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			config_json TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_by TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, policy_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS policy_violations (
			tenant_id TEXT NOT NULL,
			violation_id TEXT NOT NULL,
			policy_id TEXT NOT NULL,
			event_id TEXT,
			worker_id TEXT,
			violation_type TEXT NOT NULL,
			severity TEXT NOT NULL,
			description TEXT NOT NULL,
			metadata_json TEXT,
			detected_at TEXT NOT NULL,
			resolved_at TEXT,
			resolved_by TEXT,
			resolution_note TEXT,
			PRIMARY KEY (tenant_id, violation_id),
			FOREIGN KEY (tenant_id, policy_id) REFERENCES governance_policies(tenant_id, policy_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS budgets (
			tenant_id TEXT NOT NULL,
			budget_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id TEXT,
			limit_usd REAL NOT NULL,
			alert_threshold_pct REAL NOT NULL,
			current_spend_usd REAL NOT NULL DEFAULT 0,
			period_start TEXT NOT NULL,
			period_end TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			utilization_pct REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (tenant_id, budget_id),
			FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
		);`,
	}

	for _, statement := range statements {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("%w: migrate sqlite: %v", ErrUnavailable, err)
		}
	}
	if err := r.ensureSQLiteColumns(ctx); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_token_usage_idempotency ON token_usage_events(tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL;`); err != nil {
		return fmt.Errorf("%w: migrate sqlite: %v", ErrUnavailable, err)
	}
	if _, err := r.db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_stripe_customer ON tenants(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;`); err != nil {
		return fmt.Errorf("%w: migrate sqlite: %v", ErrUnavailable, err)
	}
	if _, err := r.db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_stripe_subscription ON tenants(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;`); err != nil {
		return fmt.Errorf("%w: migrate sqlite: %v", ErrUnavailable, err)
	}
	return nil
}

func (r *SQLiteRepository) ensureSQLiteColumns(ctx context.Context) error {
	tables := map[string]map[string]string{
		"tenants": {
			"tier":                   "TEXT NOT NULL DEFAULT 'free'",
			"usage_limit_usd":        "REAL NOT NULL DEFAULT 10.00",
			"stripe_customer_id":     "TEXT",
			"stripe_subscription_id": "TEXT",
		},
		"token_usage_events": {
			"idempotency_key":  "TEXT",
			"prompt_excerpt":   "TEXT",
			"output_excerpt":   "TEXT",
			"prompt_reference": "TEXT",
			"output_reference": "TEXT",
			"fingerprint":      "TEXT",
		},
		"api_keys": {
			"role": "TEXT NOT NULL DEFAULT 'admin'",
		},
	}
	for table, columns := range tables {
		for column, definition := range columns {
			exists, err := r.sqliteColumnExists(ctx, table, column)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			if _, err := r.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
				return fmt.Errorf("%w: migrate sqlite add column %s.%s: %v", ErrUnavailable, table, column, err)
			}
		}
	}
	return nil
}

func (r *SQLiteRepository) sqliteColumnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, wrapDBErr(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return false, wrapDBErr(err)
		}
		if name == column {
			return true, nil
		}
	}
	return false, wrapDBErr(rows.Err())
}

func (r *SQLiteRepository) UpsertTenant(ctx context.Context, tenant domain.Tenant) error {
	tier := tenant.Tier
	if tier == "" {
		tier = "free"
	}
	limit := tenant.UsageLimitUSD
	if limit == 0 {
		limit = 10.0
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tenants (tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id) DO UPDATE SET
			name = excluded.name,
			tier = excluded.tier,
			usage_limit_usd = excluded.usage_limit_usd,
			stripe_customer_id = excluded.stripe_customer_id,
			stripe_subscription_id = excluded.stripe_subscription_id,
			updated_at = excluded.updated_at
	`, tenant.TenantID, tenant.Name, tier, limit, nullString(tenant.StripeCustomerID), nullString(tenant.StripeSubscriptionID), formatTime(tenant.CreatedAt), formatTime(tenant.UpdatedAt))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) GetTenant(ctx context.Context, tenantID string) (*domain.Tenant, error) {
	var t domain.Tenant
	var stripeCust, stripeSub sql.NullString
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM tenants
		WHERE tenant_id = ?
	`, tenantID).Scan(&t.TenantID, &t.Name, &t.Tier, &t.UsageLimitUSD, &stripeCust, &stripeSub, &createdAt, &updatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	if stripeCust.Valid {
		t.StripeCustomerID = stripeCust.String
	}
	if stripeSub.Valid {
		t.StripeSubscriptionID = stripeSub.String
	}
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return &t, nil
}

func (r *SQLiteRepository) GetTenantByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Tenant, error) {
	var t domain.Tenant
	var stripeCust, stripeSub sql.NullString
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM tenants
		WHERE stripe_customer_id = ?
	`, stripeCustomerID).Scan(&t.TenantID, &t.Name, &t.Tier, &t.UsageLimitUSD, &stripeCust, &stripeSub, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	if stripeCust.Valid {
		t.StripeCustomerID = stripeCust.String
	}
	if stripeSub.Valid {
		t.StripeSubscriptionID = stripeSub.String
	}
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return &t, nil
}

func (r *SQLiteRepository) GetTenantByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (*domain.Tenant, error) {
	var t domain.Tenant
	var stripeCust, stripeSub sql.NullString
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
		FROM tenants
		WHERE stripe_subscription_id = ?
	`, stripeSubscriptionID).Scan(&t.TenantID, &t.Name, &t.Tier, &t.UsageLimitUSD, &stripeCust, &stripeSub, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	if stripeCust.Valid {
		t.StripeCustomerID = stripeCust.String
	}
	if stripeSub.Valid {
		t.StripeSubscriptionID = stripeSub.String
	}
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return &t, nil
}

func (r *SQLiteRepository) ListPricingOverrides(ctx context.Context, tenantID string) ([]domain.PricePoint, error) {
	return nil, nil
}

func (r *SQLiteRepository) GetTenantCurrentMonthCost(ctx context.Context, tenantID string) (float64, error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var total sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT SUM(cost_estimate_usd)
		FROM token_usage_events
		WHERE tenant_id = ? AND occurred_at >= ?
	`, tenantID, formatTime(startOfMonth)).Scan(&total)

	if err != nil {
		return 0, wrapDBErr(err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

func (r *SQLiteRepository) DeleteTenantData(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tenants WHERE tenant_id = ?`, tenantID)
	return wrapDBErr(err)
}

func (r *SQLiteRepository) GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error) {
	var point domain.PricePoint
	var created string
	err := r.db.QueryRowContext(ctx, `
		SELECT provider, model_id, prompt_price_per_million, completion_price_per_million, created_at
		FROM tenant_pricing_overrides
		WHERE tenant_id = ? AND provider = ? AND model_id = ?
	`, tenantID, provider, modelID).Scan(&point.Provider, &point.ModelID, &point.InputCostPerMillion, &point.OutputCostPerMillion, &created)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	point.Currency = "USD"
	point.Source = "override"
	point.EffectiveFrom = parseTime(created)
	point.CachedInputCostPerMillion = point.InputCostPerMillion / 2.0
	return &point, nil
}

func (r *SQLiteRepository) SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error {
	overrideID := tenantID + ":" + point.Provider + ":" + point.ModelID
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tenant_pricing_overrides (override_id, tenant_id, provider, model_id, prompt_price_per_million, completion_price_per_million, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, provider, model_id) DO UPDATE SET
			prompt_price_per_million = excluded.prompt_price_per_million,
			completion_price_per_million = excluded.completion_price_per_million,
			created_at = excluded.created_at
	`, overrideID, tenantID, point.Provider, point.ModelID, point.InputCostPerMillion, point.OutputCostPerMillion, formatTime(time.Now().UTC()))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) SaveAPIKey(ctx context.Context, key domain.APIKey) error {
	role := normalizeRole(key.Role)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO api_keys (key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, key.KeyID, key.TenantID, key.Name, key.KeyHash, role, formatTime(key.CreatedAt), timePtrString(key.LastUsedAt), boolInt(key.IsRevoked))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error) {
	var key domain.APIKey
	var createdAt string
	var lastUsedAt sql.NullString
	var isRevoked int
	err := r.db.QueryRowContext(ctx, `
		SELECT key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked
		FROM api_keys
		WHERE key_id = ?
	`, keyID).Scan(&key.KeyID, &key.TenantID, &key.Name, &key.KeyHash, &key.Role, &createdAt, &lastUsedAt, &isRevoked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBErr(err)
	}
	key.CreatedAt = parseTime(createdAt)
	key.Role = normalizeRole(key.Role)
	if lastUsedAt.Valid {
		parsed := parseTime(lastUsedAt.String)
		key.LastUsedAt = &parsed
	}
	key.IsRevoked = isRevoked == 1
	return &key, nil
}

func (r *SQLiteRepository) UpsertTenantMember(ctx context.Context, member domain.TenantMember) error {
	now := member.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	createdAt := member.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tenant_members (tenant_id, subject_id, email, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, subject_id) DO UPDATE SET
			email = excluded.email,
			role = excluded.role,
			updated_at = excluded.updated_at
	`, member.TenantID, member.SubjectID, nullString(member.Email), normalizeRole(member.Role), formatTime(createdAt), formatTime(now))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) ListTenantMembers(ctx context.Context, tenantID string) ([]domain.TenantMember, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, subject_id, email, role, created_at, updated_at
		FROM tenant_members
		WHERE tenant_id = ?
		ORDER BY role, subject_id
	`, tenantID)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer func() { _ = rows.Close() }()
	var members []domain.TenantMember
	for rows.Next() {
		var member domain.TenantMember
		var email sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&member.TenantID, &member.SubjectID, &email, &member.Role, &createdAt, &updatedAt); err != nil {
			return nil, wrapDBErr(err)
		}
		member.Email = email.String
		member.Role = normalizeRole(member.Role)
		member.CreatedAt = parseTime(createdAt)
		member.UpdatedAt = parseTime(updatedAt)
		members = append(members, member)
	}
	return members, wrapDBErr(rows.Err())
}

func (r *SQLiteRepository) SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	metadataJSON, err := marshalNullable(event.Metadata)
	if err != nil {
		return err
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_events (tenant_id, event_id, type, actor, resource, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, event.TenantID, event.EventID, event.Type, event.Actor, nullString(event.Resource), metadataJSON, formatTime(event.Timestamp))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, event_id, type, actor, resource, metadata_json, created_at
		FROM audit_events
		WHERE tenant_id = ?
		ORDER BY created_at DESC, event_id DESC
		LIMIT ?
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer func() { _ = rows.Close() }()
	var events []domain.AuditEvent
	for rows.Next() {
		var event domain.AuditEvent
		var resource, metadata, createdAt sql.NullString
		if err := rows.Scan(&event.TenantID, &event.EventID, &event.Type, &event.Actor, &resource, &metadata, &createdAt); err != nil {
			return nil, wrapDBErr(err)
		}
		event.Resource = resource.String
		event.Timestamp = parseTime(createdAt.String)
		if metadata.Valid && metadata.String != "" {
			_ = json.Unmarshal([]byte(metadata.String), &event.Metadata)
		}
		events = append(events, event)
	}
	return events, wrapDBErr(rows.Err())
}

func (r *SQLiteRepository) SetRecommendationState(ctx context.Context, state domain.RecommendationState) error {
	now := state.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	createdAt := state.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recommendation_states (tenant_id, recommendation_id, status, actor, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, recommendation_id) DO UPDATE SET
			status = excluded.status,
			actor = excluded.actor,
			note = excluded.note,
			updated_at = excluded.updated_at
	`, state.TenantID, state.RecommendationID, state.Status, nullString(state.Actor), nullString(state.Note), formatTime(createdAt), formatTime(now))
	return wrapDBErr(err)
}

func (r *SQLiteRepository) ListRecommendationStates(ctx context.Context, tenantID string) ([]domain.RecommendationState, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, recommendation_id, status, actor, note, created_at, updated_at
		FROM recommendation_states
		WHERE tenant_id = ?
	`, tenantID)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer func() { _ = rows.Close() }()
	var states []domain.RecommendationState
	for rows.Next() {
		var state domain.RecommendationState
		var actor, note sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&state.TenantID, &state.RecommendationID, &state.Status, &actor, &note, &createdAt, &updatedAt); err != nil {
			return nil, wrapDBErr(err)
		}
		state.Actor = actor.String
		state.Note = note.String
		state.CreatedAt = parseTime(createdAt)
		state.UpdatedAt = parseTime(updatedAt)
		states = append(states, state)
	}
	return states, wrapDBErr(rows.Err())
}

func (r *SQLiteRepository) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_keys
		SET last_used_at = ?
		WHERE key_id = ?
	`, formatTime(time.Now().UTC()), keyID)
	return wrapDBErr(err)
}

func (r *SQLiteRepository) ListAPIKeys(ctx context.Context, tenantID string) ([]domain.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked
		FROM api_keys
		WHERE tenant_id = ? AND is_revoked = 0
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer rows.Close()

	var keys []domain.APIKey
	for rows.Next() {
		var key domain.APIKey
		var createdAt string
		var lastUsedAt sql.NullString
		var isRevoked int
		if err := rows.Scan(&key.KeyID, &key.TenantID, &key.Name, &key.KeyHash, &key.Role, &createdAt, &lastUsedAt, &isRevoked); err != nil {
			return nil, wrapDBErr(err)
		}
		key.CreatedAt = parseTime(createdAt)
		if lastUsedAt.Valid {
			t := parseTime(lastUsedAt.String)
			key.LastUsedAt = &t
		}
		key.IsRevoked = isRevoked == 1
		keys = append(keys, key)
	}
	return keys, wrapDBErr(rows.Err())
}

func (r *SQLiteRepository) RevokeAPIKey(ctx context.Context, tenantID, keyID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_keys
		SET is_revoked = 1
		WHERE tenant_id = ? AND key_id = ?
	`, tenantID, keyID)
	return wrapDBErr(err)
}

func (r *SQLiteRepository) upsertTenant(ctx context.Context, tx *sql.Tx, tenantID string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO tenants (tenant_id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(tenant_id) DO UPDATE SET updated_at = excluded.updated_at
	`, tenantID, tenantID, formatTime(now), formatTime(now))
	return err
}

func (r *SQLiteRepository) upsertWorker(ctx context.Context, tx *sql.Tx, tenantID, workerID, workerName string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workers (tenant_id, worker_id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, worker_id) DO UPDATE SET
			name = excluded.name,
			updated_at = excluded.updated_at
	`, tenantID, workerID, workerName, formatTime(now), formatTime(now))
	return err
}

func (r *SQLiteRepository) upsertJob(ctx context.Context, tx *sql.Tx, tenantID, jobID, workerID, taskCategory string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO jobs (tenant_id, job_id, worker_id, name, task_category, status, started_at, ended_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, job_id) DO UPDATE SET
			worker_id = excluded.worker_id,
			task_category = excluded.task_category,
			updated_at = excluded.updated_at
	`, tenantID, jobID, workerID, jobID, taskCategory, "active", nil, nil, formatTime(now), formatTime(now))
	return err
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

	if err := r.upsertTenant(ctx, tx, event.TenantID, now); err != nil {
		return wrapDBErr(err)
	}

	workerName := event.WorkerName
	if workerName == "" {
		workerName = event.WorkerID
	}
	if err := r.upsertWorker(ctx, tx, event.TenantID, event.WorkerID, workerName, now); err != nil {
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
		if err := r.upsertJob(ctx, tx, event.TenantID, event.JobID, event.WorkerID, taskCategory, now); err != nil {
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

	_, err = tx.ExecContext(ctx, `
		INSERT INTO token_usage_events (
			tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
			provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
			input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
			cost_is_degraded, cost_degraded_code, external_estimate_usd,
			external_estimate_currency, latency_ms, task_category, output_status,
			review_score, occurred_at, created_at, prompt_excerpt, output_excerpt,
			prompt_reference, output_reference, tags_json, idempotency_key, fingerprint
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			prompt_excerpt = excluded.prompt_excerpt,
			output_excerpt = excluded.output_excerpt,
			prompt_reference = excluded.prompt_reference,
			output_reference = excluded.output_reference,
			tags_json = excluded.tags_json,
			idempotency_key = excluded.idempotency_key,
			fingerprint = excluded.fingerprint
	`, event.TenantID, event.EventID, event.WorkerID, workerName, nullString(event.JobID),
		nullString(event.SessionID), nullString(event.RunID), event.Provider, event.ModelID,
		event.PromptTokens, event.CompletionTokens, event.CachedTokens, event.InputTokens,
		event.OutputTokens, event.TotalTokens, event.CostEstimateUSD, event.CostCurrency,
		boolInt(event.CostIsDegraded), nullString(event.CostDegradedCode), externalCost,
		externalCurrency, event.LatencyMs, taskCategory, string(event.OutputStatus),
		event.ReviewScore, formatTime(event.Timestamp), formatTime(now), nullString(event.PromptExcerpt),
		nullString(event.OutputExcerpt), nullString(event.PromptReference), nullString(event.OutputReference),
		tagsJSON, nullString(event.IdempotencyKey), nullString(event.Fingerprint))
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

func (r *SQLiteRepository) SaveOutputAnalysis(ctx context.Context, analysis domain.OutputAnalysis) error {
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
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO output_analyses (
			tenant_id, analysis_id, event_id, worker_id, analyzed_at,
			efficiency_score, goblin_score, issues_json, recommendations_json,
			evidence_json, degraded_json, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, analysis_id) DO UPDATE SET
			analyzed_at = excluded.analyzed_at,
			efficiency_score = excluded.efficiency_score,
			goblin_score = excluded.goblin_score,
			issues_json = excluded.issues_json,
			recommendations_json = excluded.recommendations_json,
			evidence_json = excluded.evidence_json,
			degraded_json = excluded.degraded_json
	`, analysis.TenantID, analysis.AnalysisID, analysis.EventID, analysis.WorkerID,
		formatTime(analysis.AnalyzedAt), analysis.EfficiencyScore, analysis.GoblinScore,
		string(issuesJSON), string(recsJSON), string(evidenceJSON), degradedJSON,
		formatTime(time.Now().UTC()))
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

func (r *SQLiteRepository) ListOutputAnalyses(ctx context.Context, tenantID string, limit int) ([]domain.OutputAnalysis, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, outputAnalysisSelect+`
		WHERE tenant_id = ?
		ORDER BY analyzed_at DESC, analysis_id DESC
		LIMIT ?
	`, tenantID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer func() { _ = rows.Close() }()
	return scanOutputAnalyses(rows)
}

func (r *SQLiteRepository) ListOutputAnalysesByWorker(ctx context.Context, tenantID, workerID string, limit int) ([]domain.OutputAnalysis, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, outputAnalysisSelect+`
		WHERE tenant_id = ? AND worker_id = ?
		ORDER BY analyzed_at DESC, analysis_id DESC
		LIMIT ?
	`, tenantID, workerID, limit)
	if err != nil {
		return nil, wrapDBErr(err)
	}
	defer func() { _ = rows.Close() }()
	return scanOutputAnalyses(rows)
}

const tokenEventSelect = `
	SELECT tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
		provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
		input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
		cost_is_degraded, cost_degraded_code, external_estimate_usd,
		external_estimate_currency, latency_ms, task_category, output_status,
		review_score, occurred_at, created_at, prompt_excerpt, output_excerpt,
		prompt_reference, output_reference, tags_json, idempotency_key, fingerprint
	FROM token_usage_events
`

func scanTokenEvents(rows *sql.Rows) ([]domain.TokenEvent, error) {
	var events []domain.TokenEvent
	for rows.Next() {
		var event domain.TokenEvent
		var jobID, sessionID, runID, costCode, externalCurrency, promptExcerpt, outputExcerpt, promptReference, outputReference, tags, idempotencyKey, fingerprint sql.NullString
		var cost, externalCost, reviewScore sql.NullFloat64
		var costIsDegraded int
		var occurredAt, createdAt string
		if err := rows.Scan(&event.TenantID, &event.EventID, &event.WorkerID, &event.WorkerName,
			&jobID, &sessionID, &runID, &event.Provider, &event.ModelID, &event.PromptTokens,
			&event.CompletionTokens, &event.CachedTokens, &event.InputTokens, &event.OutputTokens,
			&event.TotalTokens, &cost, &event.CostCurrency, &costIsDegraded, &costCode,
			&externalCost, &externalCurrency, &event.LatencyMs, &event.TaskCategory,
			&event.OutputStatus, &reviewScore, &occurredAt, &createdAt, &promptExcerpt,
			&outputExcerpt, &promptReference, &outputReference, &tags, &idempotencyKey, &fingerprint); err != nil {
			return nil, wrapDBErr(err)
		}
		event.JobID = jobID.String
		event.SessionID = sessionID.String
		event.RunID = runID.String
		if cost.Valid {
			costCopy := cost.Float64
			event.CostEstimateUSD = &costCopy
		}
		event.CostDegradedCode = costCode.String
		event.CostIsDegraded = costIsDegraded != 0
		if externalCost.Valid {
			event.ExternalEstimate = &domain.ExternalEstimate{
				CostUSD:  externalCost.Float64,
				Currency: externalCurrency.String,
			}
		}
		event.Timestamp = parseTime(occurredAt)
		event.CreatedAt = parseTime(createdAt)
		event.PromptExcerpt = promptExcerpt.String
		event.OutputExcerpt = outputExcerpt.String
		event.PromptReference = promptReference.String
		event.OutputReference = outputReference.String
		if reviewScore.Valid {
			event.ReviewScore = &reviewScore.Float64
		}
		if tags.Valid && tags.String != "" && tags.String != "{}" {
			_ = json.Unmarshal([]byte(tags.String), &event.Tags)
		}
		event.IdempotencyKey = idempotencyKey.String
		event.Fingerprint = fingerprint.String
		events = append(events, event)
	}
	return events, wrapDBErr(rows.Err())
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
	defer func() { _ = rows.Close() }()
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
	defer func() { _ = rows.Close() }()
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
	defer func() { _ = rows.Close() }()

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
		if details.Valid && details.String != "" && details.String != "{}" {
			_ = json.Unmarshal([]byte(details.String), &signal.Details)
		}
		signals = append(signals, signal)
	}
	return signals, wrapDBErr(rows.Err())
}

const outputAnalysisSelect = `
	SELECT tenant_id, analysis_id, event_id, worker_id, analyzed_at,
		efficiency_score, goblin_score, issues_json, recommendations_json,
		evidence_json, degraded_json
	FROM output_analyses
`

func scanOutputAnalyses(rows *sql.Rows) ([]domain.OutputAnalysis, error) {
	var analyses []domain.OutputAnalysis
	for rows.Next() {
		var analysis domain.OutputAnalysis
		var analyzedAt string
		var issuesJSON, recommendationsJSON, evidenceJSON string
		var degradedJSON sql.NullString
		if err := rows.Scan(
			&analysis.TenantID,
			&analysis.AnalysisID,
			&analysis.EventID,
			&analysis.WorkerID,
			&analyzedAt,
			&analysis.EfficiencyScore,
			&analysis.GoblinScore,
			&issuesJSON,
			&recommendationsJSON,
			&evidenceJSON,
			&degradedJSON,
		); err != nil {
			return nil, wrapDBErr(err)
		}
		analysis.AnalyzedAt = parseTime(analyzedAt)
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
		if degradedJSON.Valid && degradedJSON.String != "" && degradedJSON.String != "[]" && degradedJSON.String != "null" {
			if err := json.Unmarshal([]byte(degradedJSON.String), &analysis.Degraded); err != nil {
				return nil, wrapDBErr(err)
			}
		}
		analyses = append(analyses, analysis)
	}
	return analyses, wrapDBErr(rows.Err())
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

func normalizeRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	switch r {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst, domain.RoleIngest, domain.RoleViewer:
		return r
	default:
		return domain.RoleViewer
	}
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

func (r *SQLiteRepository) DeleteOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	res, err := r.db.ExecContext(ctx, "DELETE FROM token_usage_events WHERE occurred_at < ?", formatTime(cutoff))
	if err != nil {
		return 0, wrapDBErr(err)
	}
	return res.RowsAffected()
}

func (r *SQLiteRepository) UpsertAgent(ctx context.Context, agent domain.Agent) error {
	return errors.New("not implemented")
}
func (r *SQLiteRepository) ListAgents(ctx context.Context, tenantID string) ([]domain.Agent, error) {
	return nil, errors.New("not implemented")
}
func (r *SQLiteRepository) UpsertGovernancePolicy(ctx context.Context, policy domain.GovernancePolicy) error {
	return errors.New("not implemented")
}
func (r *SQLiteRepository) ListGovernancePolicies(ctx context.Context, tenantID string) ([]domain.GovernancePolicy, error) {
	return nil, errors.New("not implemented")
}
func (r *SQLiteRepository) UpsertBudget(ctx context.Context, budget domain.Budget) error {
	return errors.New("not implemented")
}
func (r *SQLiteRepository) ListBudgets(ctx context.Context, tenantID string) ([]domain.Budget, error) {
	return nil, errors.New("not implemented")
}
