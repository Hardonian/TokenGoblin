# TokenGoblin Architecture And Spec

## Product Position

TokenGoblin is a token-efficiency review service for founders, engineering
leads, support leaders, and AI-ops teams. It helps answer where tokens are being
wasted, which workers or workflows are costly, which outputs show bloat or weak
structure, and which deterministic changes could reduce estimated spend.

The product intentionally avoids fake intelligence. V1 analysis is deterministic
and evidence-backed: token counts, configured pricing, output status, review
score when supplied by the customer, anomaly thresholds, optional bounded
prompt/output excerpts, and persisted history.

## Implemented Architecture

- Go HTTP API with JSON envelopes and tenant-scoped routes.
- SQLite default persistence with Postgres support through migrations.
- Tenant boundary via `tenant_id` filters on every repository read/write.
- API-key auth using `Bearer key_id.secret`; `x-tenant-id` remains for demo/local compatibility.
- Async ingestion queue with deterministic normalization, pricing, anomaly detection, and output analysis.
- Next dashboard that consumes real API responses and exposes empty/degraded states.
- CSV and Markdown exports scoped to the requesting tenant.

## Data Model

Core persisted entities:

- `tenants`: plan/billing-ready tenant record.
- `workers`, `jobs`: tenant-scoped attribution boundaries.
- `token_usage_events`: canonical usage ledger with provider/model, input/output/cached tokens, cost estimate, output status, optional evidence excerpts/references, and idempotency key.
- `cost_snapshots`: immutable estimate snapshot per event.
- `anomaly_signals`: deterministic spend/token/latency/failure/unknown-pricing signals.
- `output_analyses`: deterministic score, issues, recommendations, evidence, and degraded flags.
- `productivity_summaries`: cached tenant summaries.
- `tenant_pricing_overrides`, `api_keys`: SaaS control surfaces.

Indexes exist for tenant/time, tenant/worker, idempotency, anomaly time, and output-analysis worker review paths.

## API Contracts

All user-facing responses use:

```json
{
  "ok": true,
  "status": "success",
  "data": {},
  "degraded": []
}
```

Errors return structured `error.code` values and avoid stack traces.

Important routes:

- `POST /api/ingest/token-usage`
- `POST /api/ingest/token-usage/batch`
- `GET /api/dashboard/overview`
- `GET /api/dashboard/workers`
- `GET /api/dashboard/workers/{worker_id}`
- `GET /api/dashboard/output-analysis`
- `GET /api/dashboard/recommendations`
- `GET /api/dashboard/export.csv`
- `GET /api/dashboard/report.md`
- `GET /api/pricing`
- `POST /api/pricing/overrides`

## Deterministic Analysis

Cost:

- Uses input/output/cached token pricing per million tokens.
- Unknown model pricing creates `unknown_model_pricing` degraded state.
- Client-supplied `cost_estimate_usd` is ignored; callers may store untrusted `external_estimate`.

Output analysis:

- Flags output bloat, verbosity, repetition, weak structure, missing verification markers, missing prompt constraints, duplicate context risk, unnecessary tool-use evidence, and high-cost events.
- Keeps evidence snippets/metrics and recommendation strings.
- If prompt/output excerpts are absent, skipped checks are marked `insufficient_*_evidence`.

Recommendations:

- Routing recommendations compare accepted-output cost per task category across observed models.
- Estimated savings are labeled as estimates and include evidence count/confidence.
- No latency or quality parity is claimed unless evidence exists.

## Production Status

Implemented:

- Tenant-scoped storage access.
- API-key authentication foundation.
- Deterministic ingestion, pricing, anomaly, productivity, output-analysis, and routing logic.
- Graceful degraded states for storage unavailability, unknown pricing, no data, and missing evidence.
- Demo seed and smoke verification.
- Export-ready CSV and Markdown reports.

Partially implemented:

- Quota checks use tenant usage limits, but full plan gating/admin UI is not present.
- Billing columns and a Stripe sync skeleton exist, but Stripe webhooks are not configured.
- Postgres migrations exist, but RLS policy enforcement is not included.

Planned:

- Verified Stripe webhooks using raw body handling in the correct runtime.
- RBAC/SSO and organization role management.
- Supabase RLS policy pack.
- Recommendation acceptance state UI, recurring review runs, and scheduled reports.
