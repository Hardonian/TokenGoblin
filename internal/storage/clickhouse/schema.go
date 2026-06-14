package clickhouse

// SchemaDDL contains the ClickHouse table definitions for TokenGoblin
const SchemaDDL = `
-- Events table: raw token usage events
CREATE TABLE IF NOT EXISTS token_events (
    id              String,
    tenant_id       String,
    user_id         String,
    model           String,
    feature         String,
    prompt_tokens   UInt64,
    completion_tokens UInt64,
    total_tokens    UInt64,
    cost_usd        Float64,
    timestamp       DateTime64(3),
    prompt_fingerprint String,
    metadata        Map(String, String),
    ingestion_time  DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (tenant_id, timestamp, model, feature)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Aggregated usage table: pre-computed rollups for fast queries
CREATE TABLE IF NOT EXISTS usage_aggregates (
    tenant_id       String,
    period_start    DateTime,
    period_end      DateTime,
    model           String,
    feature         String,
    request_count   UInt64,
    total_tokens    UInt64,
    prompt_tokens   UInt64,
    completion_tokens UInt64,
    cost_usd        Float64,
    unique_users    UInt64,
    granularity     String, -- 'hour', 'day', 'week', 'month'
    computed_at     DateTime64(3) DEFAULT now64()
) ENGINE = SummingMergeTree(request_count, total_tokens, prompt_tokens, completion_tokens, cost_usd, unique_users)
PARTITION BY toDate(period_start)
ORDER BY (tenant_id, model, feature, period_start, granularity)
TTL period_start + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;

-- Anomalies table: detected cost anomalies
CREATE TABLE IF NOT EXISTS anomalies (
    id              String,
    tenant_id       String,
    anomaly_type    String, -- 'spike', 'drift', 'zombie'
    severity        String, -- 'low', 'medium', 'high', 'critical'
    description     String,
    timestamp       DateTime64(3),
    metric_value    Float64,
    threshold       Float64,
    metadata        Map(String, String),
    acknowledged    Boolean DEFAULT false,
    acknowledged_at DateTime64(3) NULL,
    acknowledged_by String
) ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (tenant_id, timestamp, anomaly_type)
TTL timestamp + INTERVAL 1 YEAR
SETTINGS index_granularity = 8192;

-- Zombie agents table: agents with low acceptance rates
CREATE TABLE IF NOT EXISTS zombie_agents (
    agent_id        String,
    tenant_id       String,
    acceptance_rate Float64,
    total_cost      Float64,
    total_requests  UInt64,
    last_activity   DateTime64(3),
    recommendation  String, -- 'quarantine', 'investigate', 'alert'
    detected_at     DateTime64(3) DEFAULT now64(),
    resolved_at     DateTime64(3) NULL,
    resolved_by     String
) ENGINE = ReplacingMergeTree(detected_at)
ORDER BY (tenant_id, agent_id)
TTL detected_at + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Prompt fingerprints table: for deduplication
CREATE TABLE IF NOT EXISTS prompt_fingerprints (
    fingerprint     String,
    tenant_id       String,
    model           String,
    first_seen      DateTime64(3) DEFAULT now64(),
    last_seen       DateTime64(3) DEFAULT now64(),
    occurrence_count UInt64,
    total_tokens    UInt64,
    total_cost_usd  Float64,
    sample_prompt   String,
    status          String DEFAULT 'active' -- 'active', 'deduplicated', 'archived'
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (tenant_id, fingerprint, model)
TTL first_seen + INTERVAL 1 YEAR
SETTINGS index_granularity = 8192;

-- Materialized view for hourly aggregates from raw events
CREATE MATERIALIZED VIEW IF NOT EXISTS usage_aggregates_hourly
ENGINE = SummingMergeTree()
PARTITION BY toDate(period_start)
ORDER BY (tenant_id, model, feature, period_start)
AS SELECT
    tenant_id,
    toStartOfHour(timestamp) AS period_start,
    toStartOfHour(timestamp) + INTERVAL 1 HOUR AS period_end,
    model,
    feature,
    count() AS request_count,
    sum(total_tokens) AS total_tokens,
    sum(prompt_tokens) AS prompt_tokens,
    sum(completion_tokens) AS completion_tokens,
    sum(cost_usd) AS cost_usd,
    uniqExact(user_id) AS unique_users,
    'hour' AS granularity
FROM token_events
GROUP BY tenant_id, period_start, period_end, model, feature;

-- Materialized view for daily aggregates
CREATE MATERIALIZED VIEW IF NOT EXISTS usage_aggregates_daily
ENGINE = SummingMergeTree()
PARTITION BY toDate(period_start)
ORDER BY (tenant_id, model, feature, period_start)
AS SELECT
    tenant_id,
    toStartOfDay(timestamp) AS period_start,
    toStartOfDay(timestamp) + INTERVAL 1 DAY AS period_end,
    model,
    feature,
    count() AS request_count,
    sum(total_tokens) AS total_tokens,
    sum(prompt_tokens) AS prompt_tokens,
    sum(completion_tokens) AS completion_tokens,
    sum(cost_usd) AS cost_usd,
    uniqExact(user_id) AS unique_users,
    'day' AS granularity
FROM token_events
GROUP BY tenant_id, period_start, period_end, model, feature;

-- Index for faster anomaly queries
-- ALTER TABLE anomalies ADD INDEX idx_tenant_severity (tenant_id, severity) TYPE set(100) GRANULARITY 1;
`