# Pull Request: v0.4.0 Full Productization & LLM Spend Audit Service

## Overview
This PR transitions TokenGoblin from v0.4.0 Release Candidate to **Full Productization** ready for live revenue, incorporating enterprise SaaS hardening and introducing the first-dollar **LLM Spend Audit** productized service.

Branch: `feat/productization-and-spend-audit`  
Target: `main`

---

## Key Deliverables & Changes

### 1. SaaS & Multi-Tenant Hardening (Part A)
- **Stripe Webhook Idempotency & Lifecycle**:
  - Implemented event replay idempotency in `internal/billing/stripe.go` using tenant audit event records (`stripe_event:<id>`). Duplicate webhooks safely return `already_processed` without duplicate side-effects.
  - Subscription failure handling: automatically downgrades delinquent tenants to `TierFree` and resets ingestion limits upon `invoice.payment_failed`.
  - Signature verification: raw body HMAC verification (`webhook.ConstructEvent`) and 64KB body caps via `http.MaxBytesReader` in `internal/api/handlers.go`.
- **Tenant Isolation Integration Test Suite** (`internal/api/tenant_isolation_test.go`):
  - Proves cross-tenant protection: Tenant A cannot read/revoke Tenant B's API keys, view B's spend/billing, read B's audit logs, reset B's data, or spoof B's `tenant_id` in ingestion payloads.
  - Windows SQLite compatibility: all database instances cleaned up via `t.Cleanup(func() { _ = repo.Close() })`.
- **Degraded Path Coverage & Complexity Reduction**:
  - Refactored `Detect` cyclomatic complexity in `internal/anomaly/detector.go` from 35 down to $<10$ by extracting modular detection helper functions.
  - Added test suites for missing costs, zero timestamps, corrupt formats, and empty slices (`detector_degraded_test.go`, `engine_degraded_test.go`).
  - Standardized error wrapping via `errors.Is(err, pgx.ErrNoRows)` in Postgres and SQLite repositories.

### 2. LLM Spend Audit Offering (Part B)
- **Audit CLI Generator** (`cmd/audit/main.go`):
  - Ingests raw CSV and JSON/JSONL usage exports from OpenAI, Anthropic, Claude, or TokenGoblin.
  - Authoritative pricing computation via `internal/cost.LoadRegistry`.
  - Deterministic breakdowns: per-team, per-model, and per-feature token attribution and top burn drivers.
  - Four savings playbooks ($30\%–50\%$ net monthly savings):
    1. 70/20/10 Tiered Model Routing
    2. Prompt & Prefix Caching
    3. Model Right-Sizing
    4. Output Length & max_tokens Controls
  - Deterministic output generation in `--out`: `audit_report.html`, `audit_report.md`, and `breakdown.csv`.
- **Executive Audit Landing Page** (`frontend/src/app/audit/page.tsx` + `layout.tsx`):
  - High-converting sales page featuring $1,500 Standard and $3,000 Enterprise packages with a 3x ROI money-back guarantee.
  - SEO metadata, canonical links, and OpenGraph/Twitter social cards.
  - Lead capture form connected upstream to `/api/contact`.
  - Global navigation link added to `Header.tsx`.
- **Operational Runbook** (`docs/AUDIT_RUNBOOK.md`):
  - Comprehensive 72-hour operational runbook detailing intake, data sanitization, CLI execution, playbook calibration, and executive debrief steps.

### 3. Developer & Operational Tooling
- **CI / CD Pipeline** (`.github/workflows/ci.yml`):
  - Compiles both `token-goblin` server and `token-goblin-audit` CLI binaries and uploads as workflow artifacts.
  - Runs `golangci-lint` with 5m timeout.
  - Runs Go unit/integration tests with `-race` and coverage tracking.
  - Runs frontend lint, typecheck, unit tests (`npm run test:ci`), and Next.js standalone build.
  - Tests and compiles TypeScript and Python SDKs.
- **GoReleaser Configuration** (`.goreleaser.yml`):
  - Dual binary build targets: `token-goblin` and `token-goblin-audit` across Linux, macOS, and Windows (amd64, arm64).
- **Deployment Manifests** (`fly.toml` & `deploy/fly/fly.toml`):
  - Production Fly.io deployment config with `/healthz` and `/readyz` probes and persistent SQLite data volume.
- **SDK Packaging**:
  - `sdks/python`: Added `pyproject.toml` and `README.md`.
  - `sdks/typescript`: Added `README.md` and type definitions.
- **SEO & Metadata**:
  - `frontend/src/app/sitemap.ts` and `frontend/src/app/robots.ts` configured for all public routes.

---

## Verification Matrix

| Verification Gate | Command | Status |
|---|---|---|
| Go Build | `go build ./...` | ✅ Exit 0 |
| Go Tests | `go test -count=1 ./...` | ✅ All 18 packages pass fresh |
| Go Code Formatting | `gofmt -l .` | ✅ 0 unformatted files |
| Frontend Typecheck | `cd frontend && npm run typecheck` | ✅ 0 errors |
| Frontend Lint | `cd frontend && npm run lint` | ✅ 0 warnings |
| Frontend Tests | `cd frontend && npm run test:ci` | ✅ 30/30 tests pass |
| Frontend Production Build | `cd frontend && npm run build` | ✅ 25 static/dynamic routes generated |
| Stripe Dry Run | `python3 scripts/setup_stripe_prices.py --dry-run` | ✅ Exit 0 |
| Audit Generator E2E | `cmd/audit --input ./examples/sample_llm_export.csv --out ./out/sample_audit` | ✅ Generated HTML/MD/CSV with 38.5% savings |
| Smoke Test | `go run ./cmd/smoke` | ✅ Ingestion, workers, anomalies verified |

---

## Live Deployment Checklist (Human Execution)
1. Run `python3 scripts/setup_stripe_prices.py` with live `sk_live_...` credentials to create production Stripe products and prices.
2. In Stripe Dashboard, configure the webhook endpoint (`https://app.tokengoblin.com/api/stripe/webhook`) listening to:
   - `checkout.session.completed`
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.payment_succeeded`
   - `invoice.payment_failed`
3. Generate `TG_INTERNAL_WEBHOOK_SECRET` (`openssl rand -hex 32`) and populate in backend and Vercel environments.
4. Deploy backend container via `scripts/deploy.sh fly` (or Railway) with `TG_ENV=production`.
5. Deploy `frontend/` to Vercel and populate `NEXT_PUBLIC_TG_API_BASE`, `NEXT_PUBLIC_STRIPE_PRICE_PRO`, and `NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE`.
6. Perform a $1 live test transaction on production.
