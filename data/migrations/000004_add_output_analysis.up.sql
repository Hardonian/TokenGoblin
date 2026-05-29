ALTER TABLE token_usage_events
ADD COLUMN IF NOT EXISTS prompt_excerpt TEXT,
ADD COLUMN IF NOT EXISTS output_excerpt TEXT,
ADD COLUMN IF NOT EXISTS prompt_reference TEXT,
ADD COLUMN IF NOT EXISTS output_reference TEXT;

CREATE TABLE IF NOT EXISTS output_analyses (
    tenant_id TEXT NOT NULL,
    analysis_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    worker_id TEXT NOT NULL,
    analyzed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    efficiency_score INTEGER NOT NULL,
    goblin_score INTEGER NOT NULL,
    issues_json JSONB NOT NULL,
    recommendations_json JSONB NOT NULL,
    evidence_json JSONB NOT NULL,
    degraded_json JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (tenant_id, analysis_id),
    FOREIGN KEY (tenant_id, event_id) REFERENCES token_usage_events(tenant_id, event_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_output_analyses_tenant_analyzed ON output_analyses (tenant_id, analyzed_at DESC);
CREATE INDEX IF NOT EXISTS idx_output_analyses_tenant_worker ON output_analyses (tenant_id, worker_id, analyzed_at DESC);
