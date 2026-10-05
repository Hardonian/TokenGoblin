# TokenGoblin deploy map (single source of truth)

| Path | What | Status |
| --- | --- | --- |
| `workers/wrangler.toml` | Go API on Cloudflare Workers + D1 (`tokengoblin-api`) | LIVE |
| `../frontend/wrangler.toml` | Next.js frontend on the edge via OpenNext (`tokengoblin-web`) | deploy-ready (`npm run deploy:edge`) |
| `../docker-compose.yml` | Full local stack: Postgres + Redis + backend + frontend (+ demo seeder, optional ClickHouse telemetry mirror) | ready |
| `d1/schema.sql` | D1 schema for the Workers API | live reference |
| `cloudflare/MIGRATION.md` | Vercel→Pages + R2 cost/latency rationale + provisioning steps | docs |
| `landing/`, `frontend/` (here) | Static landing assets | as-is |

Rules:
- The frontend wrangler config lives at `frontend/wrangler.toml` (the OpenNext
  build looks for it there). A duplicate under `deploy/cloudflare/` was removed
  on 2026-10-05 — do not recreate a second copy; edit the file above.
- Never commit secrets (Stripe keys, webhook secrets). Use `wrangler secret put`
  / environment variables.
- The Stripe webhook handler must always verify the signature over the RAW
  request body (see `frontend/src/app/api/stripe/webhook/route.ts`).
- ClickHouse is an OPTIONAL secondary analytics mirror, enabled with
  `TG_CLICKHOUSE_ADDR=clickhouse:9000` (compose: `--profile telemetry`).
  Postgres is the source of truth: events persist there first, mirror writes
  are best-effort with a bounded timeout, and a mirror failure is a logged
  warning — never a failed ingest. The seed tool verifies BOTH stores and
  reports the mirrored count.
- Production cutovers (`wrangler deploy`, `npm run deploy:edge`) are deliberate
  human decisions — config being ready is not permission to flip traffic.