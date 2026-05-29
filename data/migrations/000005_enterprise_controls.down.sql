DROP TABLE IF EXISTS recommendation_states;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS tenant_members;

ALTER TABLE api_keys
DROP COLUMN IF EXISTS role;
