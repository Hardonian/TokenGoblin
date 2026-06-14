# TokenGoblin Stateless API Tier + ClickHouse Cluster Design

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           EXTERNAL CLIENTS                                   │
│  (SDKs, Webhooks, Grafana, Custom dashboards, 3rd party integrations)       │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        EDGE / LOAD BALANCER                                  │
│         (Cloudflare / AWS ALB / NGINX / Caddy - TLS termination)             │
│         Rate limiting, DDoS protection, Request routing                      │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    STATELESS API TIER (Horizontal Scale)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │ API Pod #1   │  │ API Pod #2   │  │ API Pod #3   │  │ API Pod #N   │    │
│  │              │  │              │  │              │  │              │    │
│  │ - Auth       │  │ - Auth       │  │ - Auth       │  │ - Auth       │    │
│  │ - Validate   │  │ - Validate   │  │ - Validate   │  │ - Validate   │    │
│  │ - Transform  │  │ - Transform  │  │ - Transform  │  │ - Transform  │    │
│  │ - Enqueue    │  │ - Enqueue    │  │ - Enqueue    │  │ - Enqueue    │    │
│  │ - Respond    │  │ - Respond    │  │ - Respond    │  │ - Respond    │    │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘    │
│         │                │                │                │                │
│         └────────────────┼────────────────┼────────────────┘                │
│                          ▼                                                  │
│              ┌───────────────────────┐                                     │
│              │  INGESTION MESSAGE    │                                     │
│              │  BUS (Kafka / NATS /  │                                     │
│              │  Redis Streams)       │                                     │
│              └───────────┬───────────┘                                     │
└──────────────────────────│──────────────────────────────────────────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌────────────┐ ┌────────────┐ ┌────────────┐
       │ Ingestion  │ │ Ingestion  │ │ Ingestion  │
       │ Worker #1  │ │ Worker #2  │ │ Worker #N  │
       │            │ │            │ │            │
       │ - Batch    │ │ - Batch    │ │ - Batch    │
       │ - Dedupe   │ │ - Dedupe   │ │ - Dedupe   │
       │ - Write CH │ │ - Write CH │ │ - Write CH │
       └────────────┘ └────────────┘ └────────────┘
              │            │            │
              └────────────┼────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌────────────┐ ┌────────────┐ ┌────────────┐
       │ ClickHouse │ │ ClickHouse │ │ ClickHouse │
       │ Shard #1   │ │ Shard #2   │ │ Shard #N   │
       │ (Replica)  │ │ (Replica)  │ │ (Replica)  │
       └────────────┘ └────────────┘ └────────────┘
              │            │            │
              └────────────┼────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌────────────┐ ┌────────────┐ ┌────────────┐
       │  Query     │ │  Query     │ │  Query     │
       │  Router    │ │  Router    │ │  Router    │
       │  (CH Proxy)│ │  (CH Proxy)│ │  (CH Proxy)│
       └────────────┘ └────────────┘ └────────────┘
              │            │            │
              └────────────┼────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │   API QUERY LAYER       │
              │  (Stateless, cached)    │
              └─────────────────────────┘
```

## Stateless API Tier Design

### API Service Responsibilities
1. **Authentication & Authorization** - JWT validation, API key verification, tenant isolation
2. **Request Validation** - Schema validation, payload size limits, sanity checks
3. **Transformation** - Normalize input to internal event format
4. **Enqueueing** - Async write to message bus (fire-and-forget for ingestion)
5. **Response** - Immediate 202 Accepted for ingestion, 200 for queries
6. **Observability** - Request logging, metrics, tracing headers

### API Endpoints

```
POST   /v1/events/ingest          # Single event ingestion (202)
POST   /v1/events/batch           # Batch ingestion (202)
GET    /v1/tenants/:id/costs      # Cost queries with filters
GET    /v1/tenants/:id/costs/summary
GET    /v1/tenants/:id/costs/by-model
GET    /v1/tenants/:id/costs/by-feature
GET    /v1/tenants/:id/anomalies
GET    /v1/tenants/:id/zombie-agents
GET    /v1/tenants/:id/exports    # Export job management
POST   /v1/tenants/:id/exports
GET    /v1/healthz                # Liveness
GET    /v1/readyz                 # Readiness (CH connectivity)
```

### Horizontal Scaling Strategy

| Component | Scaling Trigger | Max Replicas | Scale-down Delay |
|-----------|----------------|--------------|------------------|
| API Pods | CPU > 70%, RPS > 5k/pod, Queue depth > 1000 | 50 | 5 min |
| Ingestion Workers | Message bus lag > 10k, Batch wait > 5s | 20 | 2 min |
| ClickHouse Shards | Storage > 70%, Query latency p99 > 5s | 10 | Manual |
| Query Routers | Concurrent queries > 200, CPU > 60% | 20 | 3 min |

### Stateless Guarantees
- **No local state** - All state in ClickHouse, Redis, or message bus
- **Session affinity NOT required** - Any pod handles any request
- **Idempotent ingestion** - Client-generated event IDs enable deduplication
- **Graceful degradation** - Query cache serves stale data if CH unavailable

## ClickHouse Cluster Design

### Cluster Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                    CLICKHOUSE CLUSTER                            │
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   SHARD 1   │    │   SHARD 2   │    │   SHARD N   │         │
│  │  (tenant %) │    │  (tenant %) │    │  (tenant %) │         │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘         │
│         │                  │                  │                 │
│    ┌────┴────┐         ┌────┴────┐         ┌────┴────┐         │
│    │ Replica │         │ Replica │         │ Replica │         │
│    │   #1    │         │   #1    │         │   #1    │         │
│    └────┬────┘         └────┬────┘         └────┬────┘         │
│         │                   │                   │              │
│    ┌────┴────┐         ┌────┴────┐         ┌────┴────┐         │
│    │ Replica │         │ Replica │         │ Replica │         │
│    │   #2    │         │   #2    │         │   #2    │         │
│    └────┬────┘         └────┬────┘         └────┬────┘         │
│         │                   │                   │              │
│    ┌────┴────┐         ┌────┴────┐         ┌────┴────┐         │
│    │ Replica │         │ Replica │         │ Replica │         │
│    │   #3    │         │   #3    │         │   #3    │         │
│    └─────────┘         └─────────┘         └─────────┘         │
│                                                                  │
│  Distributed Table: token_events_distributed                    │
│  Sharding Key: tenant_id (consistent hashing)                   │
│  Replication: 3x (quorum writes, async reads)                   │
└─────────────────────────────────────────────────────────────────┘
```

### Table Schema (Optimized for Time-Series)

```sql
-- Main events table (per shard)
CREATE TABLE token_events (
    id                  String,
    tenant_id           String,
    user_id             String,
    model               LowCardinality(String),
    feature             LowCardinality(String),
    prompt_tokens       UInt64,
    completion_tokens   UInt64,
    total_tokens        UInt64,
    cost_usd            Decimal(18, 8),
    timestamp           DateTime64(3),
    prompt_fingerprint  String,
    metadata            Map(String, String),
    ingestion_time      DateTime64(3) DEFAULT now64()
) ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/token_events', '{replica}')
PARTITION BY toDate(timestamp)
ORDER BY (tenant_id, timestamp, model, feature)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Distributed table for queries
CREATE TABLE token_events_distributed AS token_events
ENGINE = Distributed(cluster, default, token_events, tenant_id);
```

### Sharding Strategy

| Approach | Pros | Cons | Best For |
|----------|------|------|----------|
| `tenant_id` | Tenant isolation, even distribution for many tenants | Hot tenants cause skew | Multi-tenant SaaS |
| `tenant_id + model` | Better distribution for model-heavy tenants | More complex | Model analytics focus |
| Hash-based | Uniform distribution | Cross-shard queries slower | Very high volume |

**Recommendation**: `tenant_id` with consistent hashing + virtual nodes (100 per physical shard)

### Replication & Consistency

```
Write Path (Quorum):
  Client → API → Ingestion Worker → CH Shard Leader
                    ↓
            2 of 3 replicas ACK (quorum=2)
                    ↓
               Return 202 Accepted

Read Path:
  Client → API → Query Router → CH Replica (round-robin)
                           ↓
                    Stale reads OK for analytics
                    (max 1-2s replication lag)
```

### Capacity Planning

| Metric | Small | Medium | Large |
|--------|-------|--------|-------|
| Events/day | 10M | 100M | 1B+ |
| Tenants | 100 | 1,000 | 10,000 |
| Shards | 3 | 6 | 12+ |
| Replicas/shard | 2 | 3 | 3 |
| Storage/shard | 500GB | 2TB | 10TB+ |
| RAM/node | 64GB | 128GB | 256GB |
| CPU/node | 16 cores | 32 cores | 64 cores |

### Compression & Retention

```sql
-- Column compression
SETTINGS 
    compression_codec = 'ZSTD(3)',  -- Best ratio for text
    compress_by_default = true;

-- Tiered storage (hot/warm/cold)
ALTER TABLE token_events 
MODIFY SETTING storage_policy = 'tiered';

-- TTL for auto-cleanup
TTL timestamp + INTERVAL 90 DAY DELETE,
    timestamp + INTERVAL 30 DAY TO DISK 'cold',
    timestamp + INTERVAL 7 DAY TO DISK 'warm';
```

### Materialized Views for Pre-Aggregation

```sql
-- Hourly rollups (auto-maintained)
CREATE MATERIALIZED VIEW usage_hourly
ENGINE = SummingMergeTree()
PARTITION BY toDate(period_start)
ORDER BY (tenant_id, model, feature, period_start)
AS SELECT
    tenant_id,
    toStartOfHour(timestamp) AS period_start,
    model,
    feature,
    count() AS request_count,
    sum(total_tokens) AS total_tokens,
    sum(prompt_tokens) AS prompt_tokens,
    sum(completion_tokens) AS completion_tokens,
    sum(cost_usd) AS cost_usd,
    uniqExact(user_id) AS unique_users
FROM token_events
GROUP BY tenant_id, period_start, model, feature;

-- Anomaly detection view
CREATE MATERIALIZED VIEW anomaly_candidates
ENGINE = MergeTree()
PARTITION BY toDate(window_start)
ORDER BY (tenant_id, window_start, metric)
AS SELECT
    tenant_id,
    toStartOfHour(timestamp) AS window_start,
    'cost_per_token' AS metric,
    sum(cost_usd) / nullIf(sum(total_tokens), 0) AS value,
    count() AS sample_size
FROM token_events
GROUP BY tenant_id, window_start
HAVING sample_size > 100;
```

## Ingestion Pipeline with Backpressure

### Flow Control Layers

```
1. EDGE (LB)          → Rate limit per tenant (token bucket)
2. API PODS           → Circuit breaker, request queue, 202 response
3. MESSAGE BUS        → Kafka/NATS with consumer groups
4. INGESTION WORKERS  → Batch accumulation, retry with backoff
5. CLICKHOUSE         → Async inserts, quorum writes
```

### Backpressure Signals

| Layer | Signal | Action |
|-------|--------|--------|
| Edge | 429 Too Many Requests | Client backs off |
| API | Queue depth > 80% | Return 503, pause ingestion |
| Message Bus | Consumer lag > 100k | Scale workers, alert |
| Workers | CH write latency > 5s | Reduce batch size, increase workers |
| ClickHouse | Merge lag, disk space | Scale shards, TTL cleanup |

### Deduplication Strategy

```go
// At ingestion worker level
func (w *Worker) Deduplicate(batch []Event) []Event {
    // 1. Fingerprint-based dedup in worker memory (LRU cache)
    // 2. ClickHouse prompt_fingerprints table for cross-worker dedup
    // 3. Insert with ON CONFLICT DO NOTHING (CH 23.8+)
    // 4. Emit dedup metrics for monitoring
}
```

## Query Layer Design

### Query Router (ClickHouse Proxy)

```
                     ┌─────────────────┐
                     │   Query Router  │
                     │  (chproxy /     │
                     │   custom Go)    │
                     └────────┬────────┘
                              │
           ┌──────────────────┼──────────────────┐
           ▼                  ▼                  ▼
      ┌─────────┐        ┌─────────┐        ┌─────────┐
      │ Shard 1 │        │ Shard 2 │        │ Shard N │
      │ Replica │        │ Replica │        │ Replica │
      └────┬────┘        └────┬────┘        └────┬────┘
           │                  │                  │
           └──────────────────┼──────────────────┘
                              ▼
                     ┌─────────────────┐
                     │  Result Merge   │
                     │  (parallel)     │
                     └────────┬────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │  Response Cache │
                     │  (Redis, 30s TTL)│
                     └─────────────────┘
```

### Query Optimization

| Query Type | Strategy |
|------------|----------|
| Tenant cost summary | Pre-aggregated `usage_aggregates` table |
| Time-range costs | Partition pruning + materialized views |
| Model/feature breakdown | LowCardinality columns + secondary indexes |
| Anomalies | Dedicated `anomalies` table + MV |
| Exports | Async job → object storage → signed URL |

### Caching Strategy

```
Layer 1: API Response Cache (Redis)
  - Key: tenant:endpoint:filters:time_range
  - TTL: 30s (cost queries), 5m (exports)
  - Invalidation: On new ingestion for tenant

Layer 2: ClickHouse Query Cache
  - ENABLE_QUERY_CACHE = 1
  - Cache TTL: 60s
  - Max entries: 10000

Layer 3: Materialized View Auto-Refresh
  - Hourly MVs: real-time on insert
  - Daily MVs: scheduled refresh at 00:15 UTC
```

## Deployment Architecture

### Kubernetes Manifests (Conceptual)

```yaml
# API Deployment (HPA)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tokengoblin-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tokengoblin-api
  template:
    spec:
      containers:
      - name: api
        image: tokengoblin/api:v1.2.0
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2000m"
            memory: "2Gi"
        env:
        - name: CH_CLUSTER
          value: "clickhouse-cluster"
        - name: KAFKA_BROKERS
          value: "kafka:9092"
---
# HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tokengoblin-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tokengoblin-api
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "5000"
```

### ClickHouse Keeper (Coordination)

```
3-node ClickHouse Keeper ensemble for:
- Replica coordination
- Distributed DDL execution
- Schema versioning
- Mutual exclusion for TTL merges
```

## Monitoring & Alerting

### Key Metrics

| Category | Metrics |
|----------|---------|
| **Ingestion** | events_received_total, events_inserted_total, ingestion_latency_p99, batch_size_avg, backpressure_rejections |
| **API** | request_duration_p99, request_rate, error_rate, queue_depth, circuit_breaker_state |
| **ClickHouse** | query_duration_p99, merge_lag, replication_lag, disk_usage, memory_usage, parts_count |
| **Business** | cost_per_tenant, anomaly_count, zombie_agent_count, export_jobs_completed |

### Critical Alerts

```yaml
# PrometheusRule
groups:
- name: tokengoblin-critical
  rules:
  - alert: IngestionBackpressureCritical
    expr: ingestion_backpressure_level > 2
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Ingestion pipeline under critical backpressure"

  - alert: ClickHouseReplicationLagHigh
    expr: clickhouse_replication_lag_seconds > 30
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "ClickHouse replication lag > 30s"

  - alert: APIErrorRateHigh
    expr: rate(api_errors_total[5m]) / rate(api_requests_total[5m]) > 0.05
    for: 3m
    labels:
      severity: critical
    annotations:
      summary: "API error rate > 5%"
```

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
- [ ] ClickHouse cluster provisioning (3 shards, 3 replicas)
- [ ] Schema deployment + materialized views
- [ ] Basic API pod with health endpoints
- [ ] Ingestion pipeline with Kafka/NATS
- [ ] CI/CD for API deployments

### Phase 2: Resilience (Week 3-4)
- [ ] Backpressure controller implementation
- [ ] Circuit breaker integration
- [ ] Retry logic with exponential backoff
- [ ] Deduplication pipeline
- [ ] Load testing (100k events/sec target)

### Phase 3: Query Layer (Week 5-6)
- [ ] Query router (chproxy or custom)
- [ ] Redis response caching
- [ ] Export job framework
- [ ] Cost query API endpoints
- [ ] Anomaly/zombie agent APIs

### Phase 4: Production Hardening (Week 7-8)
- [ ] Multi-region ClickHouse replication
- [ ] Disaster recovery runbooks
- [ ] Cost optimization (tiered storage, compression)
- [ ] Security audit (encryption, auth, network policies)
- [ ] Documentation & runbooks

## Cost Optimization

| Technique | Estimated Savings |
|-----------|-------------------|
| ZSTD compression | 60-70% storage reduction |
| Tiered storage (SSD→HDD→S3) | 40% cost reduction for cold data |
| Materialized views | 90% query latency reduction |
| Partition pruning | 50% scan reduction |
| LowCardinality columns | 30% storage for enum-like fields |
| TTL auto-cleanup | Eliminates manual cleanup ops |

---

**Next Steps**: Begin Phase 1 with ClickHouse cluster provisioning and schema deployment.