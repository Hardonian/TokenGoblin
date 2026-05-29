ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'admin';

CREATE TABLE IF NOT EXISTS tenant_members (
    tenant_id TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    email TEXT,
    role TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, subject_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_role ON tenant_members(tenant_id, role);

CREATE TABLE IF NOT EXISTS audit_events (
    tenant_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    type TEXT NOT NULL,
    actor TEXT NOT NULL,
    resource TEXT,
    metadata_json JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created ON audit_events(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS recommendation_states (
    tenant_id TEXT NOT NULL,
    recommendation_id TEXT NOT NULL,
    status TEXT NOT NULL,
    actor TEXT,
    note TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, recommendation_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);
