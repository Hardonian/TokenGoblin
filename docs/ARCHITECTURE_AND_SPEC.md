# TokenGoblin: Architecture & Product Spec

## 1. Product Spec

**Product Thesis**  
TokenGoblin is an enterprise-grade operational intelligence microservice designed to bring radical transparency to AI token usage, cost efficiency, and worker productivity. In a market flooded with "AI theater," TokenGoblin cuts through the noise by providing deterministic, actionable insights into which AI workers are burning tokens, which tasks are driving costs, and where inefficient routing occurs. It is the definitive audit trail and cost-policy engine for the AI-enabled enterprise.

**Target Users**
- AI/ML Engineering Leads (managing token burn and vendor sprawl)
- FinOps / RevOps (tracking unit economics of AI features)
- Product Managers (measuring the ROI and productivity of AI workers)

**Core Jobs-to-be-Done**
1. Track and attribute token costs to specific workers, sessions, and jobs.
2. Identify anomalous token usage, run-away loops, or unexpected cost spikes.
3. Measure true AI worker productivity (output quality/volume vs. token burn).
4. Recommend efficient model routing based on historical cost-to-productivity data.

**MVP Scope**
- Ingestion API for standard token usage events (OpenAI, Anthropic, Gemini).
- In-memory/fast-persistence pricing and cost calculation engine.
- Worker/Session/Job data model mapping token burns to business units.
- Basic anomaly detection (e.g., rate limits, sudden cost spikes).
- Simple productivity summary API and unauthenticated internal dashboard.

**Explicit Non-Goals**
- Replacing application monitoring (APM) tools like Datadog.
- Serving as a generic proxy or API gateway (we only ingest event metadata, not the raw prompt contents).
- Complex workflow orchestration.
- Generating "fake productivity claims" or subjective AI output grading.

**Enterprise Expansion Path**
- RBAC and strict tenant isolation.
- Audit logging, SSO, and compliance reporting (SOC2 readiness).
- Custom vendor pricing overrides and negotiated contract ingestion.
- Cross-vendor model routing recommendations based on historical performance.

---

## 2. System Architecture

**MVP Architecture**

1. **Ingestion Layer:**
   - A high-throughput, low-latency REST/gRPC API.
   - Accepts asynchronous usage events containing `worker_id`, `session_id`, `job_id`, `model`, `prompt_tokens`, `completion_tokens`.
   - Fire-and-forget capable with local buffering to prevent slowing down the client application.

2. **Pricing/Cost Engine:**
   - Centralized registry of model costs per 1k/1M tokens (updated dynamically or via config).
   - Translates raw token counts into normalized fiat costs ($USD).
   - Supports degraded states (e.g., if pricing config fails, defaults to safe fallbacks and flags the event).

3. **Worker/Session/Job Model:**
   - Relational mapping defining the hierarchy: `Worker` (the AI agent) -> `Session` (the interaction loop) -> `Job` (the specific task).
   - Ensures costs are cleanly attributable at every level.

4. **Anomaly Engine:**
   - Stream processing (or async worker) analyzing ingestion streams.
   - Triggers alerts for: single-request token limits exceeded, sudden spike in `Worker` burn rate, repetitive failed jobs.

5. **Productivity Summary Layer:**
   - Aggregation service that calculates cost-per-successful-job, average token-per-turn, and task completion rates.
   
6. **Dashboard/API Layer:**
   - Read-only API serving aggregated metrics and timeseries data.
   - Simple frontend consuming these endpoints for visual debugging.

7. **Export/Report Layer:**
   - Scheduled tasks generating CSV/JSON summaries for FinOps billing reconciliation.

**Billing-Ready & Tenant Isolation Boundaries:**
- Every ingested event requires a `tenant_id`.
- Database schemas/tables enforce tenant separation via Row-Level Security (RLS) or strict `tenant_id` WHERE clauses on all reads/writes.
- Exports are strictly scoped to the requesting tenant context.

**Degraded States:**
- If the database is unreachable, the ingestion API writes to an in-memory or disk-backed queue (no dropped usage events, no hard 500s to the client).
- If the pricing engine fails, ingestion continues, and costs are retroactively calculated.

---

## 3. Data Contracts

### Token Usage Event Ingest
```json
{
  "tenant_id": "tnt_01HGW...",
  "worker_id": "wrk_summarizer_v2",
  "session_id": "ses_987654321",
  "job_id": "job_112233",
  "model_id": "gemini-1.5-pro",
  "timestamp": "2026-05-26T17:23:31Z",
  "usage": {
    "prompt_tokens": 1250,
    "completion_tokens": 400,
    "total_tokens": 1650
  },
  "metadata": {
    "latency_ms": 1200,
    "status": "success"
  }
}
```

### Cost Summary Response
```json
{
  "tenant_id": "tnt_01HGW...",
  "time_range": {
    "start": "2026-05-01T00:00:00Z",
    "end": "2026-05-26T00:00:00Z"
  },
  "total_cost_usd": 142.50,
  "top_models": [
    {"model_id": "gemini-1.5-pro", "cost_usd": 100.00},
    {"model_id": "claude-3-opus", "cost_usd": 42.50}
  ],
  "top_workers": [
    {"worker_id": "wrk_summarizer_v2", "cost_usd": 80.00}
  ]
}
```

### Worker Breakdown Response
```json
{
  "worker_id": "wrk_summarizer_v2",
  "total_jobs": 1500,
  "total_cost_usd": 80.00,
  "avg_cost_per_job_usd": 0.053,
  "anomaly_score": 0.02,
  "efficiency_rating": "optimal"
}
```

### Anomaly Signal
```json
{
  "anomaly_id": "anm_556677",
  "tenant_id": "tnt_01HGW...",
  "worker_id": "wrk_rogue_agent",
  "severity": "high",
  "trigger": "velocity_spike",
  "description": "Worker exceeded 50,000 tokens in under 60 seconds.",
  "timestamp": "2026-05-26T17:25:00Z"
}
```

### Degraded/Error Response
```json
{
  "status": "accepted_degraded",
  "message": "Event buffered. Pricing service unavailable for immediate calculation.",
  "event_id": "evt_pending_999"
}
```

---

## 4. Acceptance Criteria

Codex implementation must pass the following milestone checklist:

- [ ] **Correctness:** Token ingestion correctly parses standard formats and calculates total cost within $0.0001 precision based on the configured pricing table.
- [ ] **Tenant Isolation:** A query for Tenant A must mathematically never return data for Tenant B. Unit tests must prove cross-tenant leakage is impossible.
- [ ] **Graceful Degradation:** If the DB is locked or pricing service is down, the `/ingest` endpoint still returns `202` (Accepted) and buffers locally. NO hard 500s on ingest.
- [ ] **Cost Calculation Truth:** Cost logic natively handles disparate model pricing (e.g., prompt tokens costing less than completion tokens).
- [ ] **Test/Build Verification:** 90%+ coverage on ingestion, routing, and cost calculation logic.
- [ ] **No Fake Productivity Claims:** The API calculates objective math (cost-per-job, tokens-per-second). It does not hallucinate arbitrary "quality" scores without explicit metadata input.
- [ ] **Demo Readiness:** A script exists to simulate 1,000 events across 3 workers and generates a valid Cost Summary Response.

---

## 5. Moat Strategy

TokenGoblin compounds defensibility through operational reliance and proprietary data gravity.

**Why we win after 90 days:**
- **Proprietary Usage History:** We hold the unique mapping of *business value* (Jobs/Sessions) to *vendor cost*. A generic proxy only knows tokens; we know that `Worker B` costs 3x more to accomplish `Job X` than `Worker A`.
- **Worker-Level Productivity Memory:** We track the decay or improvement of AI agent performance over time.
- **Routing Recommendation History:** With 90 days of data, TokenGoblin can say, "Routing this task to Model Y instead of Model X will save $4,000/mo with zero latency penalty."
- **Anomaly Fingerprinting:** We build a behavioral signature of what a "runaway agent" looks like, preventing massive billing shocks before AWS/OpenAI even send an invoice.
- **Enterprise Reporting/Audit Trail:** FinOps becomes reliant on our end-of-month CSVs for department chargebacks. Switching costs become massive because the entire billing workflow depends on our schemas.

---

## 6. Pricing & Packaging

**Solo / Builder ($0 - Free Tier)**
- 100,000 token events/mo.
- 7-day data retention.
- Up to 3 active workers.
- Basic API access, no exports.

**Team ($49/mo)**
- 5,000,000 token events/mo.
- 90-day data retention.
- Unlimited workers.
- Slack anomaly alerts.
- CSV/JSON exports.
- Standard dashboard access.

**Enterprise (Custom, starting at $499/mo)**
- Unlimited token events (volume-based tiering).
- 1+ year data retention.
- Strict RBAC and SSO.
- Custom pricing ingestion (negotiated vendor rates).
- Cross-tenant unified billing reports.
- Compliance audit logs.
- Dedicated support.

---

## 7. Parallel Handoff to Codex

Codex owns the implementation of this specification. 

**Endpoints Codex Should Build Now:**
- `POST /api/v1/ingest` (Accepts the Token Usage Event contract)
- `GET /api/v1/summary` (Returns the Cost Summary Response)
- `GET /api/v1/workers/:id` (Returns the Worker Breakdown)

**Models Codex Should Implement Now:**
- `Tenant`
- `Worker`
- `Session` / `Job`
- `TokenEvent` (Immutable ledger of usage)
- `PricingConfig` (Maps model ID to prompt/completion cost)

**Tests Codex Should Run:**
- Tenant isolation tests (asserting `tenant_A` cannot query `tenant_B`).
- Precision math tests for cost calculation.
- Load testing the ingestion endpoint to ensure sub-50ms response times.

**What Must Be Marked Staged/Planned (DO NOT BUILD YET):**
- Enterprise SSO / SAML integration.
- Complex ML-based anomaly prediction (stick to threshold-based anomalies for now).
- External billing system (Stripe) integration.

**What Must NOT Be Claimed Yet:**
- SOC2 Compliance (we build for it, but don't claim it until audited).
- "AI-driven quality grading" (we only measure token mechanics and cost, not output sentiment).

---

## 8. Final Review Rubric

When Codex marks implementation complete, the PR/Codebase will be reviewed against this checklist:

- [ ] **Route Smoke Tests:** Can I POST 10 events to `/ingest` and immediately GET accurate math from `/summary`?
- [ ] **Tenant Tests:** Did Codex implement a hard `tenant_id` filter on every database query?
- [ ] **Empty-State Tests:** Does the dashboard/API handle 0 ingested events gracefully without throwing dividing-by-zero errors?
- [ ] **Anomaly Test Data:** Can the system detect a simulated runaway worker script?
- [ ] **Cost Formula Tests:** Are costs calculated with arbitrary precision (e.g., Decimal/Float64) to handle fractions of a cent correctly?
- [ ] **Docs-vs-Code Truth Check:** Do the implemented JSON responses exactly match the Data Contracts in this document?
- [ ] **Deployment Readiness:** Are Dockerfiles, environment variable templates (`.env.example`), and build scripts present and working?
