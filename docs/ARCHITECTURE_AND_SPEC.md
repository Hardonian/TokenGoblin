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
- API-key roles with route-level RBAC for write-heavy and administrative actions.
- Async ingestion queue with deterministic normalization, pricing, anomaly detection, and output analysis.
- Next dashboard that consumes real API responses and exposes empty/degraded states.
- CSV and Markdown exports scoped to the requesting tenant.
- Persisted recommendation states, tenant-member records, and audit events for review history.
- Stripe webhook acknowledgement in the Next.js Node runtime with raw-body signature verification.

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
- `tenant_members`: persisted membership/role registry for enterprise access review.
- `audit_events`: tenant-scoped operational trail for imports, exports, admin actions, and recommendation decisions.
- `recommendation_states`: accepted/rejected/implemented state for deterministic routing recommendations.

Indexes exist for tenant/time, tenant/worker, idempotency, anomaly time, output-analysis worker review paths, tenant members, audit events, and recommendation states.

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
- `POST /api/dashboard/recommendations/{recommendation_id}/status`
- `GET /api/audit/events`
- `GET /api/tenant/members`
- `POST /api/tenant/members`
- `GET /api/dashboard/export.csv`
- `GET /api/dashboard/report.md`
- `GET /api/pricing`
- `POST /api/pricing/overrides`
- `POST /api/stripe/webhook` in the Next.js app, with `runtime = "nodejs"` and raw-body signature verification.

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
- Decisions are persisted separately from the deterministic recommendation so a team can track acceptance, rejection, and implementation without altering the evidence model.

## Production Status

Implemented:

- Tenant-scoped storage access.
- API-key authentication foundation.
- API-key roles and route-level RBAC for ingestion, pricing, reset/seed, and recommendation decisions.
- Deterministic ingestion, pricing, anomaly, productivity, output-analysis, and routing logic.
- Persisted tenant members, audit events, and recommendation decision states.
- Graceful degraded states for storage unavailability, unknown pricing, no data, and missing evidence.
- Demo seed and smoke verification.
- Export-ready CSV and Markdown reports.
- Verified Stripe webhook acknowledgement route in the Next.js Node runtime.

Partially implemented:

- Quota checks use tenant usage limits, but full plan gating/admin UI is not present.
- Billing columns exist, and verified Stripe webhook acknowledgement exists, but subscription lifecycle writes are not connected to tenant billing state.
- Postgres migrations exist, but RLS policy enforcement is not included.
- Tenant members are persisted, but external identity-provider sync and SSO login are not configured.

Planned:

- Stripe subscription lifecycle processing after verified webhook acknowledgement.
- SSO admin surfaces and identity-provider group sync.
- Supabase RLS policy pack.
- Recurring review runs and scheduled report delivery.
