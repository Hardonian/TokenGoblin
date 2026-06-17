# TokenGoblin Release Status

v0.4.0: PRODUCTION-READY CODEBASE
- ✅ Go backend: Ingestion, billing, anomaly detection, executive dashboards
- ✅ Next.js frontend: Pricing, billing portal, dashboard, auth
- ✅ Stripe integration: Checkout sessions, billing portal, webhook handling
- ✅ Anomaly detection: Spend/token/latency/velocity spikes, repeated failures
- ✅ Self-serve signup: /api/tenant/register → API key → ingest → dashboard
- ✅ Multi-tier pricing: Free ($0), Pro ($29/mo), Enterprise ($99/mo)
- ✅ Deployment scripts: Fly.io, Railway, Stripe price setup

BLOCKERS FOR LIVE REVENUE:
1. **Live Stripe Price IDs** — Run `scripts/setup_stripe_prices.py` with live key
2. **Production deployment** — Run `scripts/deploy.sh fly` (or railway)
3. **Stripe webhook** — Configure in Stripe Dashboard after deploy
4. **Vercel frontend** — Deploy frontend, set NEXT_PUBLIC_TG_API_BASE

VERIFICATION COMMANDS:
- go build ./cmd/server ✓
- go test ./... ✓
- pnpm --filter frontend build ✓
- python3 scripts/setup_stripe_prices.py --dry-run ✓

STATUS: CODE COMPLETE — NEEDS LIVE STRIPE + DEPLOY

NEXT ACTIONS FOR FIRST $1K MRR:
1. Run setup_stripe_prices.py with sk_live_ key → get price IDs
2. Run deploy.sh fly → get production API URL
3. Configure Stripe webhook → production_URL/api/v1/webhooks/stripe
4. Deploy frontend to Vercel → set NEXT_PUBLIC_TG_API_BASE
5. Launch: Show HN + Indie Hackers + AI agent Discords (per GTM_PLAN.md)
6. Target: 1000 visitors/day → 1.2 Pro/day → $35/day → $1K MRR by week 4