CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_stripe_customer
ON tenants(stripe_customer_id)
WHERE stripe_customer_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_stripe_subscription
ON tenants(stripe_subscription_id)
WHERE stripe_subscription_id IS NOT NULL;

CREATE OR REPLACE FUNCTION tg_current_tenant_id()
RETURNS TEXT
LANGUAGE SQL
STABLE
AS $$
    SELECT NULLIF(current_setting('app.tenant_id', true), '')
$$;

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE workers ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE token_usage_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE cost_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE anomaly_signals ENABLE ROW LEVEL SECURITY;
ALTER TABLE productivity_summaries ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_pricing_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE output_analyses ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE recommendation_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenants_tenant_isolation ON tenants;
CREATE POLICY tenants_tenant_isolation ON tenants
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS workers_tenant_isolation ON workers;
CREATE POLICY workers_tenant_isolation ON workers
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS jobs_tenant_isolation ON jobs;
CREATE POLICY jobs_tenant_isolation ON jobs
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS token_usage_events_tenant_isolation ON token_usage_events;
CREATE POLICY token_usage_events_tenant_isolation ON token_usage_events
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS cost_snapshots_tenant_isolation ON cost_snapshots;
CREATE POLICY cost_snapshots_tenant_isolation ON cost_snapshots
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS anomaly_signals_tenant_isolation ON anomaly_signals;
CREATE POLICY anomaly_signals_tenant_isolation ON anomaly_signals
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS productivity_summaries_tenant_isolation ON productivity_summaries;
CREATE POLICY productivity_summaries_tenant_isolation ON productivity_summaries
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS tenant_pricing_overrides_tenant_isolation ON tenant_pricing_overrides;
CREATE POLICY tenant_pricing_overrides_tenant_isolation ON tenant_pricing_overrides
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS output_analyses_tenant_isolation ON output_analyses;
CREATE POLICY output_analyses_tenant_isolation ON output_analyses
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS tenant_members_tenant_isolation ON tenant_members;
CREATE POLICY tenant_members_tenant_isolation ON tenant_members
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS audit_events_tenant_isolation ON audit_events;
CREATE POLICY audit_events_tenant_isolation ON audit_events
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

DROP POLICY IF EXISTS recommendation_states_tenant_isolation ON recommendation_states;
CREATE POLICY recommendation_states_tenant_isolation ON recommendation_states
FOR ALL
USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

-- API keys intentionally receive no tenant client policy. Service/database-owner
-- connections can manage them, while direct Supabase client roles are denied by
-- RLS unless a future migration grants a narrower audited policy.
