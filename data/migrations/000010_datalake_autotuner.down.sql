DROP INDEX IF EXISTS idx_token_usage_is_exported;
ALTER TABLE token_usage_events DROP COLUMN IF EXISTS is_exported;
DROP TABLE IF EXISTS tuning_profiles;
