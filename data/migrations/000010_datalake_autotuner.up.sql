CREATE TABLE tuning_profiles (
	tenant_id TEXT PRIMARY KEY REFERENCES tenants(tenant_id) ON DELETE CASCADE,
	aggressiveness REAL NOT NULL DEFAULT 1.0,
	ignored_keywords TEXT NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE token_usage_events ADD COLUMN is_exported BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_token_usage_is_exported ON token_usage_events (is_exported);
