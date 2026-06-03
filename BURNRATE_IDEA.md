# Next Million-Dollar App: "BurnRate"

## The Problem

Startup founders are terrified of one thing: running out of money. Yet most track their burn rate in spreadsheets that are updated weekly (at best). By the time they realize they have 4 months of runway instead of 6, it's too late to act.

## The Product

**BurnRate** — Real-time SaaS burn rate monitor with Slack/Discord alerts.

Connect Stripe (or Paddle) → see your exact runway in real-time → get alerts when burn changes.

## Why This Wins

1. **Pain is extreme** = founders will pay immediately to not think about this
2. **Simple to build** = 3 days to MVP
3. **Clear value prop** = "Know your runway, always"
4. **Viral in startups** = every founder knows other founders
5. **High retention** = once connected, they never disconnect

## Feature Set (MVP — 3 days)

```
Day 1: Stripe integration + basic dashboard
- Connect Stripe OAuth
- Pull MRR, expenses, cash balance
- Calculate runway = cash / monthly burn
- Simple dashboard: runway months, burn rate, MRR

Day 2: Alerts
- Slack/Discord webhook integration
- Alert when burn rate increases >10%
- Alert when runway drops below 3 months
- Daily summary digest

Day 3: Polish + launch
- Clean UI (Tailwind)
- Signup/auth (Clerk or similar)
- Pricing page
- Deploy
```

## Pricing

| Tier | Price | Features |
|------|-------|----------|
| Free | $0 | 1 Stripe account, basic dashboard |
| Pro | $19/mo | Alerts, multiple accounts, team access |
| Scale | $49/mo | API access, custom integrations, priority support |

## Revenue Projection

- Month 1: 50 signups, 5 Pro = $95/mo
- Month 3: 500 signups, 50 Pro = $950/mo
- Month 6: 2000 signups, 200 Pro + 20 Scale = $4,780/mo
- Month 12: 10000 signups, 1000 Pro + 100 Scale = $23,900/mo

**Path to $1M ARR:** 4000 Pro subscribers (achievable in 18-24 months with content marketing)

## Competitive Landscape

| Competitor | Price | Weakness |
|-----------|-------|----------|
| Runway (app) | $49/mo | Complex, over-engineered |
| Baremetrics | $59/mo | Expensive, too many features |
| Spreadsheets | Free | Manual, error-prone, stale |

**Our edge:** Simpler, cheaper, real-time alerts.

## Tech Stack

- **Backend:** Go (fast, cheap to host)
- **Frontend:** Next.js + Tailwind
- **Auth:** Clerk (free tier)
- **Payments:** Stripe (of course)
- **Hosting:** Fly.io ($5/mo to start)
- **Database:** SQLite → Postgres when scaling

## Go-to-Market

1. **Week 1:** Build MVP, launch on HN/IndieHackers
2. **Week 2:** Content — "I built a tool that tells startups when they'll run out of money"
3. **Week 3:** Partner with startup accelerators (YC, Techstars) — offer free to their portfolio
4. **Week 4:** Twitter/X campaign targeting #startuplife, #buildinpublic

## Why This Could Be $1M+

- **Recurring revenue:** Monthly subscriptions, high retention
- **Upsell path:** Free → Pro → Scale → Enterprise (custom)
- **Expansion:** Add expense tracking, revenue forecasting, investor reporting
- **Acquisition target:** Baremetrics, ChartMogul, or Stripe itself could acquire

## Risks

| Risk | Mitigation |
|------|------------|
| Stripe API changes | Abstract behind interface, monitor changelog |
| Low willingness to pay | Free tier hooks them, alerts create urgency |
| Competition | Move fast, own the "simple" positioning |

## Next Steps

1. [ ] Validate: Post on IndieHackers "Would you pay $19/mo for real-time runway tracking?"
2. [ ] Build: 3-day sprint
3. [ ] Launch: HN + Twitter + IndieHackers
4. [ ] Iterate: Talk to first 10 users, double down on what they love
