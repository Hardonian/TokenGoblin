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

API: http://localhost:8080  
Dashboard: http://localhost:3000

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

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | No | SQLite path (default: `file:token_goblin.db`) or Postgres DSN |
| `STRIPE_SECRET_KEY` | For billing | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | For billing | Stripe webhook signing secret |
| `STRIPE_PRICE_PRO` | For billing | Stripe Price ID for Pro plan |
| `STRIPE_PRICE_ENTERPRISE` | For billing | Stripe Price ID for Enterprise plan |
| `TG_INTERNAL_WEBHOOK_SECRET` | For billing | Internal secret for Stripe webhook forwarding |
| `PORT` | No | API server port (default: 8080) |
| `NEXT_PUBLIC_TG_API_BASE` | Yes (frontend) | API base URL |
| `NEXT_PUBLIC_STRIPE_PRICE_PRO` | For billing | Stripe Price ID for Pro (frontend) |
| `NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE` | For billing | Stripe Price ID for Enterprise (frontend) |

## API Reference

### Authentication
All API routes (except `/api/tenant/register`) require either:
- `x-tenant-id` header (for development/demo)
- `Authorization: Bearer <api_key>` (for production)

### Ingestion
```
POST /api/ingest/token-usage
POST /api/ingest/token-usage/batch
```

### Dashboard
```
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
```
POST /api/tenant/register          — Create account (public)
POST /api/billing/checkout         — Create Stripe Checkout session
POST /api/billing/portal           — Create Stripe Customer Portal session
GET  /api/billing/status           — Get current billing status
POST /api/stripe/webhook           — Stripe webhook (server-side)
```

### Tenant Management
```
GET    /api/tenant/members
POST   /api/tenant/members
POST   /api/dashboard/recommendations/{id}/status
DELETE /api/dashboard/reset
```

## Pricing

| Plan | Price | Events/mo | Features |
|------|-------|-----------|----------|
| Free | $0 | 10K | Dashboard, CSV export |
| Pro | $29 | 100K | + Output analysis, Goblin Score, recommendations |
| Enterprise | $99 | Unlimited | + Audit trail, RBAC, custom pricing, SLA |

## Architecture

- **Backend:** Go 1.25, SQLite (dev) / Postgres (prod), Stripe integration
- **Frontend:** Next.js 16, React 19, Tailwind CSS 4
- **Deployment:** Docker multi-stage build, single container

## License
MIT
