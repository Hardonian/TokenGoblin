# TokenGoblin — AI Spend & Token-Efficiency Observability

<!-- BEGIN: REPO HERO -->
![TokenGoblin — hero generated locally on the GPU stack](assets/repo-hero.png)
<!-- END: REPO HERO -->

Track, analyze, and optimize AI token spending across your autonomous agent workforce.

TokenGoblin provides deep, deterministic token observability, anomaly detection (runaway loops, context padding, cache misses), multi-tier model routing, and automated spend audits for engineering and FinOps teams.

---

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+
- Stripe account (for SaaS billing features)

### 1. Clone & configure

```bash
git clone https://github.com/Hardonian/TokenGoblin.git
cd TokenGoblin
cp .env.example .env
# Edit .env with your Stripe and database keys
```

### 2. Run with Docker (recommended)

```bash
docker compose up --build
```

API: <http://localhost:8080>  
Dashboard: <http://localhost:3000>

### 3. Run locally (development)

**Backend:**

```bash
go run ./cmd/server
```

**Frontend:**

```bash
cd frontend
npm install
npm run dev
```

---

## LLM Spend Audit Offering

TokenGoblin includes an executive-grade **LLM Spend Audit Generator** CLI and landing page (`/audit`):

```bash
# Build the audit generator CLI
go build -o ./bin/audit ./cmd/audit

# Run an audit on any OpenAI, Anthropic, or generic CSV/JSON usage export
./bin/audit \
  --input ./examples/sample_llm_export.csv \
  --out ./out/audit_result \
  --tenant "Acme Corporation"
```

The generator produces:
- `audit_report.html`: Self-contained, responsive dark-mode executive dashboard.
- `audit_report.md`: Complete markdown report for Notion, GitHub, and Slack.
- `breakdown.csv`: Tabular breakdown by model, team, and feature for financial analysts.

See the [Audit Runbook](docs/AUDIT_RUNBOOK.md) for full 72-hour service delivery procedures.

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `TG_ENV` | Optional | Set to `production` to enforce strict API key auth and disable demo tenant bypass. |
| `PORT` / `TG_ADDR` | Optional | Server listen port or host:port (default: `8080`). |
| `TG_DB_DSN` | Required (Prod) | PostgreSQL DSN (`postgres://...`). Production startup fails closed if omitted. |
| `TG_DB_PATH` | Optional | SQLite path for local/staging (default: `./data/tokengoblin.sqlite`). |
| `TG_REDIS_ADDR` | Optional | Redis host:port for distributed rate limiting and pricing cache. |
| `STRIPE_SECRET_KEY` | Required (Billing) | Stripe secret key (`sk_live_...` or `sk_test_...`). |
| `STRIPE_WEBHOOK_SECRET` | Required (Billing) | Stripe webhook signing secret (`whsec_...`). |
| `STRIPE_PRICE_PRO` | Required (Billing) | Stripe Price ID for Pro plan (backend). |
| `STRIPE_PRICE_ENTERPRISE` | Required (Billing) | Stripe Price ID for Enterprise plan (backend). |
| `TG_INTERNAL_WEBHOOK_SECRET` | Required (Billing) | Shared secret for internal Next.js to Go backend webhook forwarding. |
| `NEXT_PUBLIC_TG_API_BASE` | Required (Frontend) | Backend API base URL (e.g. `http://localhost:8080` or `https://api.yourdomain.com`). |
| `NEXT_PUBLIC_STRIPE_PRICE_PRO` | Required (Frontend) | Stripe Price ID for Pro plan (UI checkout). |
| `NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE` | Required (Frontend) | Stripe Price ID for Enterprise plan (UI checkout). |

> [!IMPORTANT]
> When deploying the frontend to Vercel, configure `NEXT_PUBLIC_TG_API_BASE`, `NEXT_PUBLIC_STRIPE_PRICE_PRO`, and `NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE` directly in the Vercel dashboard.

---

## Runtime & Operations

The backend runs on port `8080` by default (can be overridden via `PORT` or `TG_ADDR`).

- **Health Checks & Observability**:
  - `GET /healthz` — Liveness probe (returns `200 OK`).
  - `GET /readyz` — Readiness probe (returns `200 OK` when DB connection succeeds, `503` when unavailable).
  - `GET /metrics` — Prometheus metrics scraping endpoint.
- **Database**: In production, provide `TG_DB_DSN` for PostgreSQL. Local development and staging fallback cleanly to SQLite with WAL mode.
- **Deprecation**: Running without `TG_ENV=production` allows legacy `x-tenant-id` demo header auth without an API key. This mode is deprecated and will be removed in v1.0.

---

## API Reference

### Authentication
- Production: `Authorization: Bearer <api_key>` or `X-API-Key: <api_key>`
- Internal webhook forwarding: `Authorization: Bearer <TG_INTERNAL_WEBHOOK_SECRET>`

### Core Endpoints

```text
# Health & Metrics
GET    /healthz
GET    /readyz
GET    /metrics

# Ingestion
POST   /v1/events
POST   /v1/events/batch
POST   /api/ingest/token-usage
POST   /api/ingest/token-usage/batch

# Intelligence & Analysis (v2)
GET    /v2/intelligence/waste
GET    /v2/intelligence/prompt-graveyard
GET    /v2/intelligence/cost-leaks
GET    /v2/intelligence/zombie-agents
GET    /v2/intelligence/duplicates
GET    /v2/intelligence/hallucinations
GET    /v2/forecasts/spend
GET    /v2/executive/scorecard
GET    /v2/analytics/models

# Billing & Account
POST   /api/tenant/register
GET    /api/billing/status
POST   /api/billing/checkout
POST   /api/billing/portal
POST   /api/v1/webhooks/stripe
POST   /internal/billing/stripe-event
```

---

## SaaS Pricing

| Plan | Price | Events/mo | Features |
|---|---|---|---|
| **Scout** | $0 | 10K | Dashboard, CSV export, email support |
| **Hoarder (Pro)** | $29 | 100K | + Intelligence, spend forecasts, recommendations, zombie agent detection |
| **Warlord (Enterprise)** | $99 | Unlimited | + Audit trail, RBAC, custom pricing overrides, SLA & dedicated support |

---

## License

MIT © [Hardonian](https://github.com/Hardonian)
