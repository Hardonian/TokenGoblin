CREATE TABLE IF NOT EXISTS tuning_profiles (
	tenant_id TEXT PRIMARY KEY REFERENCES tenants(tenant_id) ON DELETE CASCADE,
	aggressiveness REAL NOT NULL DEFAULT 1.0,
	ignored_keywords TEXT NOT NULL,
	updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

ALTER TABLE token_usage_events ADD COLUMN IF NOT EXISTS is_exported BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_token_usage_is_exported ON token_usage_events (is_exported);
