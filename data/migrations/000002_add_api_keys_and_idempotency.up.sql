CREATE TABLE api_keys (
    key_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);

ALTER TABLE token_usage_events ADD COLUMN idempotency_key TEXT;
CREATE UNIQUE INDEX idx_token_usage_idempotency ON token_usage_events(tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
