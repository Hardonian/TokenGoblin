CREATE TABLE IF NOT EXISTS agents (
    tenant_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    owner_id TEXT,
    agent_type TEXT NOT NULL,
    framework TEXT,
    status TEXT NOT NULL,
    budget_usd REAL,
    budget_period TEXT,
    sla_latency_ms INTEGER,
    sla_success_rate REAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    retired_at TIMESTAMP WITH TIME ZONE,
    retirement_reason TEXT,
    PRIMARY KEY (tenant_id, agent_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS agent_performance_reviews (
    tenant_id TEXT NOT NULL,
    review_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    event_count INTEGER NOT NULL,
    total_cost_usd REAL NOT NULL,
    acceptance_rate REAL NOT NULL,
    avg_latency_ms REAL NOT NULL,
    cost_per_outcome REAL NOT NULL,
    sla_violations INTEGER NOT NULL,
    efficiency_grade TEXT NOT NULL,
    recommendation TEXT NOT NULL,
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, review_id),
    FOREIGN KEY (tenant_id, agent_id) REFERENCES agents(tenant_id, agent_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS governance_policies (
    tenant_id TEXT NOT NULL,
    policy_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config_json JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, policy_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS policy_violations (
    tenant_id TEXT NOT NULL,
    violation_id TEXT NOT NULL,
    policy_id TEXT NOT NULL,
    event_id TEXT,
    worker_id TEXT,
    violation_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    description TEXT NOT NULL,
    metadata_json JSONB,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by TEXT,
    resolution_note TEXT,
    PRIMARY KEY (tenant_id, violation_id),
    FOREIGN KEY (tenant_id, policy_id) REFERENCES governance_policies(tenant_id, policy_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS budgets (
    tenant_id TEXT NOT NULL,
    budget_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT,
    limit_usd REAL NOT NULL,
    alert_threshold_pct REAL NOT NULL,
    current_spend_usd REAL NOT NULL DEFAULT 0,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    utilization_pct REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, budget_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

-- Enable RLS for all new tables
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_performance_reviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE governance_policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE budgets ENABLE ROW LEVEL SECURITY;

-- Add RLS policies for tenant isolation
CREATE POLICY tenant_isolation_agents ON agents
    USING (tenant_id = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_agent_reviews ON agent_performance_reviews
    USING (tenant_id = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_gov_policies ON governance_policies
    USING (tenant_id = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy_violations ON policy_violations
    USING (tenant_id = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_budgets ON budgets
    USING (tenant_id = current_setting('app.current_tenant_id', true));
