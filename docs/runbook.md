# TokenGoblin :: Operational Production Runbook

> **Service:** `tokengoblin` (Go Backend + Next.js Frontend)  
> **Repository:** `Hardonian/TokenGoblin`  
> **Architecture:** Dual-Database (PostgreSQL with SQLite fallback, Redis rate limiting, Next.js frontend proxy).

---

## 1. Production Service Architecture

```mermaid
graph LR
    User[Client / Browser] -->|HTTPS| Vercel[Next.js Frontend / Vercel]
    Vercel -->|Internal API / Proxy| Backend[TokenGoblin Go Backend]
    ClientSDK[AI Agents / SDKs] -->|Bearer API Key| Backend
    Backend --> Postgres[(PostgreSQL 16)]
    Backend --> Redis[(Redis Rate Limiter)]
    Stripe[Stripe Webhooks] -->|/api/stripe/webhook| Vercel
    Vercel -->|Bearer TG_INTERNAL_WEBHOOK_SECRET| Backend
```

---

## 2. Health, Readiness & Observability Endpoints

All probes are unauthenticated on the backend listener (default `:8080`):

| Endpoint | Type | Description |
|---|---|---|
| `GET /healthz` | Liveness | Returns `200 OK` ("ok") if HTTP server is serving traffic. |
| `GET /readyz` | Readiness | Returns `200 OK` ("ready") if database connection (`repo.Ping()`) succeeds. Returns `503 Service Unavailable` if database is down. |
| `GET /metrics` | Prometheus | Standard Prometheus exposition format for metrics scraping (latency, request counts, queue depths). |

### Verification
```bash
# Liveness
curl -fsS http://localhost:8080/healthz

# Readiness
curl -fsS http://localhost:8080/readyz

# Prometheus metrics
curl -fsS http://localhost:8080/metrics | grep tokengoblin
```

---

## 3. Local & Production Deployment Procedures

### 3.1 Local Development (Docker Compose)
```bash
# Start Postgres, Redis, and TokenGoblin backend + frontend
docker compose up -d

# Verify containers are healthy
docker compose ps

# Tail backend logs
docker compose logs -f backend
```

### 3.2 Backend: Fly.io Deployment
```bash
# 1. Authenticate with Fly.io
fly auth login

# 2. Deploy backend container
fly deploy -c deploy/fly/fly.toml

# 3. View status and health
fly status
fly logs
```

### 3.3 Backend: Railway Deployment
```bash
# 1. Connect project
railway link

# 2. Deploy Dockerfile.backend
railway up --service backend

# 3. Configure production environment variables
railway variables set TG_ENV=production PORT=8080 TG_DB_DSN=$DATABASE_URL
```

### 3.4 Frontend: Vercel Deployment
```bash
# 1. Navigate to frontend
cd frontend

# 2. Deploy to Vercel production
vercel --prod \
  --build-env NEXT_PUBLIC_TG_API_BASE=https://api.tokengoblin.com \
  --build-env NEXT_PUBLIC_STRIPE_PRICE_PRO=price_... \
  --build-env NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE=price_...
```

---

## 4. Database Migrations & Administration

TokenGoblin supports PostgreSQL (via `jackc/pgx`) and SQLite (via `modernc.org/sqlite`).

### SQLite Mode (Single-Node / Small Deployment)
- Auto-migrates on startup via embedded schema definitions in `internal/storage/sqlite.go`.
- Database file is stored at `TG_DB_PATH` (defaults to `./data/tokengoblin.sqlite`).
- SQLite uses WAL mode (`journal_mode=WAL`), `busy_timeout=5000`, and `synchronous=NORMAL`.

### PostgreSQL Mode (Clustered / High-Availability)
- Set `TG_DB_DSN=postgres://user:pass@host:5432/dbname?sslmode=require`.
- Auto-migrates on startup in `internal/storage/postgres.go`.
- Connection pooling is managed automatically via `pgxpool` with health checks.

---

## 5. Security & Authentication Model

### Production Mode (`TG_ENV=production`)
- When `TG_ENV=production`, the legacy demo tenant header mode (`x-tenant-id` without an API key) is **strictly rejected with 401 Unauthorized**.
- Clients must pass `Authorization: Bearer tg_live_<key>` or `X-API-Key: tg_live_<key>`.
- Internal webhook forwarding requires `Authorization: Bearer $TG_INTERNAL_WEBHOOK_SECRET`.

---

## 6. Incident Response & Troubleshooting Runbook

| Symptom | Root Cause | Immediate Remediation |
|---|---|---|
| **`/readyz` returns 503** | Database unreachable or connection pool exhausted | Check Postgres health: `pg_isready`. Verify `TG_DB_DSN`. Inspect backend connection counts. |
| **Ingestion returns 429 Too Many Requests** | Rate limit exceeded for tenant or worker | Verify Redis connectivity. Check tenant tier in Stripe or upgrade tenant to Pro/Enterprise. |
| **Webhooks return 400 invalid_signature** | Stripe signing secret mismatch | Verify `STRIPE_WEBHOOK_SECRET` matches the active endpoint in the Stripe Dashboard. |
| **Webhooks return 401 unauthorized** | Internal proxy secret mismatch | Ensure `TG_INTERNAL_WEBHOOK_SECRET` matches in both frontend Vercel env and backend env. |
| **Degraded response returned (`status: "degraded"`)** | Primary database unavailable; read-only fallback active | Check database health; backend serves cached / fallback envelopes to protect consumer uptime. |

---

## 7. LLM Spend Audit CLI Tooling

To generate an executive spend audit for a customer from an export file:

```bash
# Build the CLI
go build -o ./bin/audit ./cmd/audit

# Run audit analysis
./bin/audit \
  --input ./data/export.csv \
  --out ./out/audit_result \
  --tenant "Acme Corporation"

# Output artifacts will be in ./out/audit_result:
# - audit_report.html (interactive dark-mode dashboard)
# - audit_report.md   (markdown summary for Slack/Notion)
# - breakdown.csv     (tabular financial data)
```
