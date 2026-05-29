DROP TABLE IF EXISTS output_analyses;

ALTER TABLE token_usage_events
DROP COLUMN IF EXISTS output_reference,
DROP COLUMN IF EXISTS prompt_reference,
DROP COLUMN IF EXISTS output_excerpt,
DROP COLUMN IF EXISTS prompt_excerpt;
