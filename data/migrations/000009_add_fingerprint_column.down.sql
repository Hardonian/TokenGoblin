DROP INDEX IF EXISTS idx_token_usage_tenant_fingerprint;
ALTER TABLE token_usage_events DROP COLUMN IF EXISTS fingerprint;