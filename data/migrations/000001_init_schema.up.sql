CREATE TABLE tenants (
    tenant_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE workers (
    tenant_id TEXT NOT NULL,
    worker_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, worker_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE TABLE jobs (
    tenant_id TEXT NOT NULL,
    job_id TEXT NOT NULL,
    worker_id TEXT NOT NULL,
    name TEXT NOT NULL,
    task_category TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, job_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, worker_id) REFERENCES workers(tenant_id, worker_id) ON DELETE CASCADE
);

CREATE TABLE token_usage_events (
    tenant_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    worker_id TEXT NOT NULL,
    worker_name TEXT NOT NULL,
    job_id TEXT,
    session_id TEXT,
    run_id TEXT,
    provider TEXT NOT NULL,
    model_id TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL,
    completion_tokens INTEGER NOT NULL,
    cached_tokens INTEGER NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,
    cost_estimate_usd DOUBLE PRECISION,
    cost_currency TEXT NOT NULL,
    cost_is_degraded BOOLEAN NOT NULL,
    cost_degraded_code TEXT,
    external_estimate_usd DOUBLE PRECISION,
    external_estimate_currency TEXT,
    latency_ms INTEGER NOT NULL,
    task_category TEXT NOT NULL,
    output_status TEXT NOT NULL,
    review_score DOUBLE PRECISION,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    tags_json JSONB,
    PRIMARY KEY (tenant_id, event_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, worker_id) REFERENCES workers(tenant_id, worker_id) ON DELETE CASCADE
);

CREATE INDEX idx_token_usage_tenant_occurred ON token_usage_events (tenant_id, occurred_at DESC);
CREATE INDEX idx_token_usage_tenant_worker ON token_usage_events (tenant_id, worker_id);

CREATE TABLE cost_snapshots (
    tenant_id TEXT NOT NULL,
    snapshot_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    model_id TEXT NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    cached_tokens INTEGER NOT NULL,
    cost_estimate_usd DOUBLE PRECISION,
    currency TEXT NOT NULL,
    is_degraded BOOLEAN NOT NULL,
    degraded_code TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, snapshot_id),
    FOREIGN KEY (tenant_id, event_id) REFERENCES token_usage_events(tenant_id, event_id) ON DELETE CASCADE
);

CREATE TABLE anomaly_signals (
    tenant_id TEXT NOT NULL,
    anomaly_id TEXT NOT NULL,
    event_id TEXT,
    worker_id TEXT,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    severity TEXT NOT NULL,
    type TEXT NOT NULL,
    description TEXT NOT NULL,
    observed_value DOUBLE PRECISION,
    threshold_value DOUBLE PRECISION,
    details_json JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, anomaly_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE INDEX idx_anomaly_tenant_detected ON anomaly_signals (tenant_id, detected_at DESC);

CREATE TABLE productivity_summaries (
    tenant_id TEXT NOT NULL,
    summary_id TEXT NOT NULL,
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    total_cost_usd DOUBLE PRECISION NOT NULL,
    total_events INTEGER NOT NULL,
    output_count INTEGER NOT NULL,
    avg_latency_ms DOUBLE PRECISION,
    anomaly_count INTEGER NOT NULL,
    cost_per_accepted_output_with_review DOUBLE PRECISION,
    summary_json JSONB NOT NULL,
    PRIMARY KEY (tenant_id, summary_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);
