# TokenGoblin Launch Completion Plan — Productization Status

Goal: make TokenGoblin fully production-ready end-to-end (backend, frontend, tests, CI, docs, Vercel deploy) with zero known compile/test regressions.

## Verified Baseline & Deliverables

- `go build ./...` passing (all packages compile clean)
- `go test -count=1 ./...` passing (all backend & CLI test suites passing fresh)
- `cd frontend && npm run lint && npm run typecheck && npm run test:ci && npm run build` passing clean (30/30 tests pass, 0 lint warnings, 23/23 static routes generated)
- `python3 scripts/setup_stripe_prices.py --dry-run` passing clean
- Full LLM Spend Audit CLI (`cmd/audit`) with deterministic HTML, Markdown, and CSV deliverables
- Executive Spend Audit landing page (`/audit`) with fixed pricing and 3x ROI guarantee

---

## Launch Block 1: Vercel Readiness ✅ COMPLETE

1. Git-to-Vercel configuration requirements documented in `README.md` and `.env.example`.
2. Frontend CI build and test pipeline operational in `.github/workflows/ci.yml`.
3. Standalone output build verified via Next.js 16 with zero warnings.

---

## Launch Block 2: Frontend Tests & Coverage ✅ COMPLETE

1. Jest and React Testing Library configured with JSDOM and modern mock lifecycle.
2. Smoke and unit test suites implemented:
   - `frontend/src/app/__tests__/page.test.tsx` (Command Center)
   - `frontend/src/app/billing/__tests__/page.test.tsx` (Billing, Tiers, Portals, Errors)
   - `frontend/src/app/pricing/__tests__/page.test.tsx` (Plans, Tiers, Checkout Redirects)
   - `frontend/src/app/audit/__tests__/page.test.tsx` (Spend Audit Hero, Packages, Lead Capture)
   - `frontend/src/app/api/tenant/register/__tests__/route.test.ts` (Registration Route)
   - `frontend/src/lib/__tests__/billing.test.ts` (Billing Utilities)

---

## Launch Block 3: Backend Safety & Billing Hardening ✅ COMPLETE

1. Backend integration test suites for `/internal/billing/stripe-event` and `/api/v1/webhooks/stripe`:
   - Authorization verification (bearer token required).
   - Replay idempotency (`already_processed` audit event verification).
   - Invoice payment failure downgrading to free tier.
   - Raw body Stripe signature verification and 64KB body caps.
2. Tenant isolation integration tests (`internal/api/tenant_isolation_test.go`):
   - Tenant cannot read another tenant's API keys.
   - Tenant cannot revoke another tenant's API keys.
   - Tenant cannot read another tenant's billing/spend data.
   - Tenant cannot read another tenant's audit logs.
   - Tenant cannot reset another tenant's data.
   - Tenant cannot ingest data with spoofed tenant IDs.
3. Degraded path test suites:
   - `internal/anomaly/detector_degraded_test.go`
   - `internal/intelligence/engine_degraded_test.go`
4. Code quality & linting:
   - Cyclomatic complexity in `internal/anomaly/detector.go` refactored into helper functions ($< 10$).
   - Dead fields removed and `errors.Is(err, pgx.ErrNoRows)` verified.
   - All Go files formatted via `gofmt`.

---

## Launch Block 4: Observability & Operational Documentation ✅ COMPLETE

1. `docs/runbook.md` reconciled with `/healthz`, `/readyz`, and `/metrics` probes.
2. Docker, Fly.io, Railway, and Vercel commands documented.
3. Incident response matrices and troubleshooting runbooks established.
4. `README.md` and `.env.example` updated with production mode (`TG_ENV=production`) and demo tenant deprecation notices.
5. `docs/AUDIT_RUNBOOK.md` detailing the 72-hour LLM Spend Audit operational delivery lifecycle.

---

## Launch Block 5: Final End-to-End Verification ✅ COMPLETE

- `go build ./...` ✅ (Exit 0)
- `go test -count=1 ./...` ✅ (All pass)
- `npm run build` ✅ (Exit 0)
- `npm run test:ci` ✅ (30/30 pass)
- `python3 scripts/setup_stripe_prices.py --dry-run` ✅ (Exit 0)
- `cmd/audit` E2E execution on sample export ✅ (Exit 0)
