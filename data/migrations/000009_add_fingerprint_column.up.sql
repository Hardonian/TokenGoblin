ALTER TABLE token_usage_events ADD COLUMN IF NOT EXISTS fingerprint TEXT;
CREATE INDEX IF NOT EXISTS idx_token_usage_tenant_fingerprint ON token_usage_events (tenant_id, fingerprint);