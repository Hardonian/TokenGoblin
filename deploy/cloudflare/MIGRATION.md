# TokenGoblin → Cloudflare (cost + latency)

## Why — and a heads-up
TokenGoblin's API is ALREADY on the edge (Cloudflare Workers + D1). The wins here are:

1. **Frontend: Vercel `iad1` → Cloudflare Pages.** Vercel is pinned to a single US region
   (`iad1` in `vercel.json`), so every non-US visitor pays a round-trip. Pages serves from 300+
   PoPs = lower latency worldwide, usually cheaper than Vercel's per-seat/GB pricing, no cold starts.

2. **Exports/datalake → R2.** `TG_DATALAKE_DIR=/data/datalake` writes JSONL exports to local disk
   (or you'd pay S3 egress). R2 = ZERO egress for downloads.

> **Redirect note:** the productization handoff told the coding agent to "deploy the frontend to
> Vercel." **This supersedes that** — target Cloudflare Pages, not Vercel, for cost + latency. If
> that agent already set up Vercel, keep it only as a fallback and cut over to Pages.

## 1) Frontend → Cloudflare Pages
Next.js with API routes (billing/checkout, stripe webhook, tenant register) needs an adapter —
**OpenNext** (`@opennextjs/cloudflare`) turns it into a Worker + edge assets (API routes become
Worker routes at the edge):

```bash
cd frontend && npm i -D @opennextjs/cloudflare
npx opennextjs-cloudflare build && npx wrangler deploy   # from frontend/ — wrangler.toml lives there
# static-only fallback: npx wrangler pages deploy frontend/.next/static --project-name tokengoblin
```
Keep `NEXT_PUBLIC_TG_API_BASE` pointing at the Workers API (`tokengoblin-api`). Stripe price IDs +
webhook secret go in env / `wrangler secret put`, never in config.

**Stripe webhook must verify the signature on the RAW body** (no re-serialization) — keep that when
the webhook moves to a Worker route.

## 2) Exports/datalake → R2
```bash
npx wrangler r2 bucket create tokengoblin-exports    # zero egress
```
Bind as `DATALAKE` (see wrangler.toml) and point the export sink at R2 (S3-compatible) instead of
`/data/datalake`.

## Latency / cost check
```bash
curl -sS -o /dev/null -w '%{http_code} %{time_total}s\n' https://<pages-domain>/
# expect lower time_total from non-US regions vs the old iad1 Vercel deploy
```

## Honest notes
- Full Next.js SSR → edge is a real migration (OpenNext); the static/CDN cut-over is the quick win.
- D1 already gives you edge reads; add KV/Cache if read-heavy queries start costing D1 reads.
- R2 meters storage + operations (not egress).
