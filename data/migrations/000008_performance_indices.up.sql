CREATE INDEX IF NOT EXISTS idx_token_usage_tenant_model ON token_usage_events (tenant_id, model_id);
CREATE INDEX IF NOT EXISTS idx_jobs_tenant_status ON jobs (tenant_id, status);
