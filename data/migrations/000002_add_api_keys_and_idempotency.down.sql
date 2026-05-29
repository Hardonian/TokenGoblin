DROP INDEX IF EXISTS idx_token_usage_idempotency;
ALTER TABLE token_usage_events DROP COLUMN IF EXISTS idempotency_key;

DROP INDEX IF EXISTS idx_api_keys_tenant;
DROP TABLE IF EXISTS api_keys;
