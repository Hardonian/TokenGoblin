ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_performance_reviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE governance_policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE budgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE tuning_profiles ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_agents ON agents;
DROP POLICY IF EXISTS tenant_isolation_agent_reviews ON agent_performance_reviews;
DROP POLICY IF EXISTS tenant_isolation_gov_policies ON governance_policies;
DROP POLICY IF EXISTS tenant_isolation_policy_violations ON policy_violations;
DROP POLICY IF EXISTS tenant_isolation_budgets ON budgets;

CREATE POLICY agents_tenant_isolation ON agents
FOR ALL USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

CREATE POLICY agent_reviews_tenant_isolation ON agent_performance_reviews
FOR ALL USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

CREATE POLICY governance_policies_tenant_isolation ON governance_policies
FOR ALL USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

CREATE POLICY policy_violations_tenant_isolation ON policy_violations
FOR ALL USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

CREATE POLICY budgets_tenant_isolation ON budgets
FOR ALL USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());

CREATE POLICY tuning_profiles_tenant_isolation ON tuning_profiles
FOR ALL USING (tenant_id = tg_current_tenant_id())
WITH CHECK (tenant_id = tg_current_tenant_id());
