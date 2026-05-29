# TokenGoblin

TokenGoblin is a deterministic token-efficiency review service for teams tracking
worker/agent token usage, output bloat, cost leaks, anomalies, and routing
opportunities.

It does not perform AI quality grading. Every visible score or recommendation is
derived from persisted usage events, configured pricing, optional bounded
prompt/output excerpts, and deterministic rules.

## Quick Start

```bash
npm install
npm run db:seed
npm run smoke
npm run dev
```

The Go API listens on `:8080` by default. The Next dashboard is in `frontend/`.

```bash
cd frontend
npm install
npm run dev
```

All API requests require either:

- `Authorization: Bearer key_id.secret`, for stored API keys
- `x-tenant-id: demo-tenant`, retained for local/demo compatibility

## Core Routes

- `POST /api/ingest/token-usage`
- `POST /api/ingest/token-usage/batch`
- `GET /api/dashboard/overview`
- `GET /api/dashboard/workers`
- `GET /api/dashboard/workers/{worker_id}`
- `GET /api/dashboard/events`
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

Example:

```bash
curl -H "x-tenant-id: demo-tenant" \
  http://localhost:8080/api/dashboard/overview
```

## Data And Analysis

Token usage events normalize provider/model, worker, run/job/session, input and
output tokens, cached tokens, latency, output status, review score, optional
bounded prompt/output excerpts, and optional external reference IDs.

Cost estimates use a configurable model pricing table with input/output/cached
token separation. Unknown model pricing is explicitly degraded and excluded from
cost totals.

Output analysis is deterministic. It can flag output bloat, repetition, weak
structure, missing verification markers, missing prompt constraints, duplicate
context risk, unnecessary tool-use evidence, and high-cost events. If text
evidence is absent, text checks are skipped and marked degraded.

## Scripts

- `npm run lint` runs `go vet ./...`
- `npm run typecheck` runs compile-only Go tests
- `npm run test` runs all Go tests
- `npm run build` builds the Go server
- `npm run db:seed` creates deterministic demo data
- `npm run smoke` seeds and verifies the execution layer
- `npm run lint --prefix frontend` lints the dashboard
- `npm run build --prefix frontend` builds the dashboard

## Environment

- `TG_ADDR`: API listen address, default `:8080`
- `TG_DB_PATH`: SQLite path, default `./data/tokengoblin.sqlite`
- `TG_DB_DSN`: Postgres DSN; when set, Postgres migrations run from `data/migrations`
- `TG_PRICING_TABLE_JSON`: JSON pricing overrides keyed by `provider:model`
- `TG_DISABLE_DEFAULT_PRICING=1`: disables bundled default pricing
- `TG_REDIS_ADDR`: optional Redis address for rate limiting/cache diagnostics
- `TG_DEMO_TENANT_ID`: tenant used by seed/smoke, default `demo-tenant`
- `NEXT_PUBLIC_TG_API_BASE`: dashboard API base, default `http://localhost:8080`
- `TG_API_BASE`: server-side API base used by the Next.js Stripe webhook forwarder
- `STRIPE_WEBHOOK_SECRET`: Stripe signing secret for the Next.js Node runtime route at `/api/stripe/webhook`
- `TG_INTERNAL_WEBHOOK_SECRET`: shared internal secret for forwarding verified Stripe events into the Go billing lifecycle endpoint
- `TG_ENV=production`: enables fail-closed deployment checks
- `TG_ALLOW_DEMO_AUTH=1`: allows `x-tenant-id` demo auth outside production only
- `TG_PLAN_FREE_LIMIT_USD`, `TG_PLAN_PREMIUM_LIMIT_USD`: deterministic quota limits applied by Stripe lifecycle events, defaults `$10` and `$100`

## Production Notes

Implemented:

- Tenant-scoped repository reads/writes
- API-key authentication with hashed secrets
- API-key roles and route-level RBAC for ingestion, pricing, reset/seed, and recommendation decisions
- Deterministic pricing, anomaly, productivity, output-analysis, and routing rules
- Persisted recommendation acceptance/rejection/implementation state
- Tenant member registry and audit/event log foundation
- CSV and Markdown tenant exports
- Verified Stripe webhook route in the Next.js Node runtime using raw request bodies
- Stripe subscription lifecycle forwarding into tenant tier, quota, customer, and subscription state
- Production fail-closed checks for Postgres DSN, internal webhook secret, and demo-auth disablement
- Supabase/Postgres RLS migration pack using `app.tenant_id` tenant policies
- SQLite schema repair for older local/demo databases
- Graceful degraded responses for unavailable storage and missing evidence

Partially implemented:

- Stripe checkout linking supports tenant metadata/client reference IDs; full hosted billing portal and checkout creation are not implemented
- RLS policies are present for Supabase/Postgres; direct client sessions must set `app.tenant_id` or use service-role/server-side access
- Tenant members are persisted for access-review and admin workflows; full SSO/identity-provider sync is not configured

Planned:

- Stripe checkout and billing portal creation
- SSO admin surfaces and identity-provider group sync
- Recurring review scheduling and report delivery
