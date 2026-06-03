# TokenGoblin GTM & Rollout Plan

## Pre-Launch Checklist

- [ ] Deploy to production (Fly.io / Railway / Render)
- [ ] Set up Stripe test mode → create Price IDs for Pro ($29) and Enterprise ($99)
- [ ] Configure Stripe webhook endpoint → `/api/stripe/webhook`
- [ ] Set `TG_INTERNAL_WEBHOOK_SECRET` (generate: `openssl rand -hex 32`)
- [ ] Set `NEXT_PUBLIC_TG_API_BASE` to production URL
- [ ] Set `NEXT_PUBLIC_STRIPE_PRICE_PRO` and `NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE`
- [ ] Test full flow: signup → ingest → dashboard → billing → upgrade
- [ ] Set up monitoring (UptimeRobot free tier)

## Week 1: Soft Launch

### Day 1-2: Deploy & Test
```bash
# Option A: Fly.io (recommended for Go)
fly launch --dockerfile Dockerfile
fly secrets set STRIPE_SECRET_KEY=sk_live_...
fly secrets set STRIPE_WEBHOOK_SECRET=whsec_...
fly secrets set TG_INTERNAL_WEBHOOK_SECRET=...
fly deploy

# Option B: Railway
# Connect GitHub repo → auto-deploy on push
```

### Day 3-4: Content
- Write "Show HN" post: "TokenGoblin — Open-source AI spend observability"
- Prepare Reddit posts for r/SaaS, r/startups, r/Artificial, r/LocalLLaMA
- Create Twitter/X thread: "We built a tool that tracks AI token costs across agents"

### Day 5-7: Launch
- Post on Hacker News (Show HN)
- Post on Indie Hackers
- Share in relevant Discord servers (AI agent communities)
- Email any beta users / personal network

## Week 2: Content Marketing

### Blog Posts (write 2-3)
1. "How We Cut Our AI Spend by 40% with TokenGoblin"
2. "The Hidden Cost of AI Agents: A Token-by-Token Analysis"
3. "Open-Source vs. Paid: Building an AI Observability Stack"

### Distribution
- Dev.to (cross-post from blog)
- Medium (cross-post)
- LinkedIn (target: CTOs, engineering managers)
- Twitter/X (thread format)

## Week 3-4: Product Hunt & Partnerships

### Product Hunt Launch
- Prepare screenshots, demo video (Loom), tagline
- Coordinate with friends for initial upvotes
- Engage with all comments

### Partnerships
- Reach out to AI agent frameworks (CrewAI, AutoGen, LangGraph)
- Offer integration guides
- Guest post on their blogs

## Conversion Funnel Optimization

```
Visitor → Pricing Page (40% of visitors)
       → Signup (10% of pricing visitors = 4% conversion)
       → Ingest Data (60% of signups = 2.4% of visitors)
       → See Value (80% of ingest = 1.9% of visitors)
       → Hit Free Limit (30% of active = 0.6% of visitors)
       → Upgrade to Pro (20% of limit-hit = 0.12% of visitors)

Target: 1000 visitors/day → 1.2 Pro signups/day → $35/day → $1050/month
```

## Pricing Strategy

| Tier | Price | Target |
|------|-------|--------|
| Free | $0 | Hook: get users ingesting data |
| Pro | $29/mo | Core revenue: small teams, startups |
| Enterprise | $99/mo | Scale: larger teams, custom needs |

**Annual discount:** 20% off (shown on pricing page toggle)

## Key Metrics Dashboard

Track weekly:
- Signups (target: 5/day by week 2, 20/day by month 2)
- Free → Pro conversion (target: 5%)
- MRR growth (target: $500 by month 1, $2000 by month 3)
- Churn (target: <10% monthly)
- API ingestion volume (proxy for engagement)

## Competitive Positioning

| Competitor | Weakness | Our Edge |
|-----------|----------|----------|
| OpenAI Usage Dashboard | OpenAI only | Multi-provider, agent-focused |
| LangSmith | Complex, expensive | Simple, affordable, self-hosted |
| Custom spreadsheets | Manual, error-prone | Automated, real-time |

## 90-Day Milestones

- [ ] Week 1: Deploy, first 10 signups
- [ ] Week 2: 50 signups, first Pro conversion
- [ ] Week 4: 200 signups, $500 MRR
- [ ] Week 8: 1000 signups, $2000 MRR
- [ ] Week 12: 3000 signups, $5000 MRR, break even on hosting costs
