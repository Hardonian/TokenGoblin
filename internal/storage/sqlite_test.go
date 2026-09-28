package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			idempotency_key TEXT,
			fingerprint TEXT,
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

func TestNormalizeRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected string
	}{
		{"owner", domain.RoleOwner, domain.RoleOwner},
		{"admin", domain.RoleAdmin, domain.RoleAdmin},
		{"analyst", domain.RoleAnalyst, domain.RoleAnalyst},
		{"ingest", domain.RoleIngest, domain.RoleIngest},
		{"viewer", domain.RoleViewer, domain.RoleViewer},
		{"unknown", "unknown", domain.RoleViewer},
		{"empty", "", domain.RoleViewer},
		{"uppercase handles gracefully", "OWNER", domain.RoleOwner},
		{"whitespace handles gracefully", " admin ", domain.RoleAdmin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeRole(tt.role))
		})
	}
}

func TestIsValidSQLIdentifier(t *testing.T) {
	assert.True(t, isValidSQLIdentifier("tenants"))
	assert.True(t, isValidSQLIdentifier("token_usage_events"))
	assert.False(t, isValidSQLIdentifier("token_usage_events; DROP TABLE users;"))
	assert.False(t, isValidSQLIdentifier("a b"))
	assert.False(t, isValidSQLIdentifier(""))
}

func TestSQLiteColumnExists_SQLInjection(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	repo, err := OpenSQLite(context.Background(), dbPath)
	require.NoError(t, err)
	defer repo.Close()

	_, err = repo.sqliteColumnExists(context.Background(), "tenants; DROP TABLE users;", "id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid table name")
}

func TestSQLiteFounderModePersistence(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenSQLite(ctx, filepath.Join(t.TempDir(), "founder.sqlite"))
	require.NoError(t, err)
	defer func() { require.NoError(t, repo.Close()) }()

	now := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, repo.UpsertTenant(ctx, domain.Tenant{
		TenantID: "tenant-a", Name: "Tenant A", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}))
	budgetUSD := 25.0
	latency := 1500
	successRate := 0.98
	require.NoError(t, repo.UpsertAgent(ctx, domain.Agent{
		AgentID: "agent-a", TenantID: "tenant-a", Name: "Reviewer",
		AgentType: domain.WorkerTypeAgent, Framework: domain.AgentFrameworkCustom,
		Status: domain.AgentStatusActive, BudgetUSD: &budgetUSD, SLALatencyMs: &latency,
		SLASuccessRate: &successRate, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.UpsertGovernancePolicy(ctx, domain.GovernancePolicy{
		PolicyID: "policy-a", TenantID: "tenant-a", Name: "Monthly ceiling",
		Type: domain.PolicyBudgetLimit, ConfigJSON: `{"limit_usd":25}`, IsActive: true,
		CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.UpsertBudget(ctx, domain.Budget{
		BudgetID: "budget-a", TenantID: "tenant-a", Name: "Default", Type: "monthly",
		ScopeType: domain.BudgetScopeTenant, LimitUSD: 25, AlertThresholdPct: 80,
		PeriodStart: now, PeriodEnd: now.AddDate(0, 1, 0), IsActive: true,
		CreatedAt: now, Status: domain.BudgetStatusHealthy,
	}))

	agents, err := repo.ListAgents(ctx, "tenant-a")
	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "Reviewer", agents[0].Name)
	assert.Equal(t, budgetUSD, *agents[0].BudgetUSD)

	policies, err := repo.ListGovernancePolicies(ctx, "tenant-a")
	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.True(t, policies[0].IsActive)

	budgets, err := repo.ListBudgets(ctx, "tenant-a")
	require.NoError(t, err)
	require.Len(t, budgets, 1)
	assert.Equal(t, 25.0, budgets[0].LimitUSD)
}
