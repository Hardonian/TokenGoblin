# TokenGoblin

Deterministic MVP execution layer for token usage ingestion, cost estimates,
tenant-scoped dashboard data, productivity primitives, anomaly detection, and a
demo seed.

## Quick Start

```bash
npm install
docker-compose up -d
npm run db:seed
npm run smoke
npm run dev
```

All API requests require an `x-tenant-id` header. The demo seed uses
`demo-tenant` by default.

```bash
curl -H "x-tenant-id: demo-tenant" http://localhost:8080/api/dashboard/overview
```

## Scripts

- `npm run lint` runs `go vet ./...`.
- `npm run typecheck` compiles packages with `go test ./... -run TestCompileOnly`.
- `npm run test` runs deterministic unit and route coverage.
- `npm run build` builds the Go server.
- `npm run db:seed` creates deterministic demo data.
- `npm run smoke` verifies the seeded execution layer.

## Implementation Notes

- Tenant scope is derived from `x-tenant-id`. A payload `tenant_id` is accepted
  only when it matches that header.
- Client-supplied costs are ignored unless sent as `external_estimate`; internal
  cost estimates come from the pricing registry.
- Unknown pricing, missing data, and database unavailability return structured
  degraded responses instead of unhandled 500s.
- Productivity scoring is deterministic in V1. No LLM quality scoring is used.
