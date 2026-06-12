# TokenGoblin — AI Spend & Token-Efficiency Observability

Track, analyze, and optimize AI token spending across your autonomous agent workforce.

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+
- Stripe account (for billing features)

### 1. Clone & configure

```bash
git clone https://github.com/Hardonian/TokenGoblin.git
cd TokenGoblin
cp .env.example .env
# Edit .env with your Stripe keys
```

### 2. Run with Docker (recommended)

```bash
docker-compose up --build
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

## Environment Variables

| Variable                              | Required           | Description                                                               |
| ------------------------------------- | ------------------ | ------------------------------------------------------------------------- |
| `DATABASE_URL`                        | Optional           | SQLite path (default: `file:token_goblin.db`) or Postgres DSN             |
| `STRIPE_SECRET_KEY`                   | Required (Billing) | Stripe secret key for backend API calls                                   |
| `STRIPE_WEBHOOK_SECRET`               | Required (Billing) | Stripe webhook signing secret to verify Stripe events                     |
| `STRIPE_PRICE_PRO`                    | Required (Billing) | Stripe Price ID for Pro plan (backend)                                    |
| `STRIPE_PRICE_ENTERPRISE`             | Required (Billing) | Stripe Price ID for Enterprise plan (backend)                             |
| `TG_INTERNAL_WEBHOOK_SECRET`          | Required (Billing) | Internal secret for Stripe webhook forwarding                             |
| `PORT`                                | Optional           | API server port (default: 8080)                                           |
| `TG_ENV`                              | Optional           | Environment mode (set to `production` to enforce API key auth)            |
| `NEXT_PUBLIC_TG_API_BASE`             | Required (Vercel)  | API base URL (e.g. `https://api.yourdomain.com`)                          |
| `NEXT_PUBLIC_STRIPE_PRICE_PRO`        | Required (Vercel)  | Stripe Price ID for Pro (frontend UI checkout)                            |
| `NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE` | Required (Vercel)  | Stripe Price ID for Enterprise (frontend UI checkout)                     |

> [!IMPORTANT]
> When deploying the frontend to Vercel, you must configure `NEXT_PUBLIC_TG_API_BASE`, `NEXT_PUBLIC_STRIPE_PRICE_PRO`, and `NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE` directly in the Vercel dashboard. Vercel ignores `vercel.json` for runtime secrets.

## Runtime & Operations

The backend runs on port `8080` by default (can be overridden via `PORT`).

- **Health/Metrics**: The backend exposes Prometheus metrics at `/metrics` which can also be used as a simple liveness probe.
- **Database**: In production, provide `TG_DB_DSN` or `DATABASE_URL` for Postgres. If omitted, it will degrade gracefully to an ephemeral SQLite database.
- **Deprecation**: Running the backend without `TG_ENV=production` enables a demo tenant mode that bypasses API key checks. This behavior is deprecated and will be removed in v1.0.

## API Reference

### Authentication

All API routes (except `/api/tenant/register`) require either:

- `x-tenant-id` header (for development/demo)
- `Authorization: Bearer <api_key>` (for production)

### Ingestion

```text
POST /api/ingest/token-usage
POST /api/ingest/token-usage/batch
```

### Dashboard

```text
GET /api/dashboard/overview
GET /api/dashboard/workers
GET /api/dashboard/workers/{worker_id}
GET /api/dashboard/events?limit=12
GET /api/dashboard/anomalies
GET /api/dashboard/output-analysis
GET /api/dashboard/recommendations
GET /api/dashboard/export.csv
GET /api/dashboard/report.md
```

### Billing

```text
POST /api/tenant/register          — Create account (public)
POST /api/billing/checkout         — Create Stripe Checkout session
POST /api/billing/portal           — Create Stripe Customer Portal session
GET  /api/billing/status           — Get current billing status
POST /api/stripe/webhook           — Stripe webhook (server-side)
```

### Tenant Management

```text
GET    /api/tenant/members
POST   /api/tenant/members
POST   /api/dashboard/recommendations/{id}/status
DELETE /api/dashboard/reset
```

## Pricing

| Plan       | Price | Events/mo | Features                                         |
| ---------- | ----- | --------- | ------------------------------------------------ |
| Free       | $0    | 10K       | Dashboard, CSV export                            |
| Pro        | $29   | 100K      | + Output analysis, Goblin Score, recommendations |
| Enterprise | $99   | Unlimited | + Audit trail, RBAC, custom pricing, SLA         |

## Architecture

- **Backend:** Go 1.25, SQLite (dev) / Postgres (prod), Stripe integration
- **Frontend:** Next.js 16, React 19, Tailwind CSS 4
- **Deployment:** Docker multi-stage build, single container

## License

MIT
