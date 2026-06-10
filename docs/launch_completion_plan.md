# TokenGoblin Launch Completion Plan — 2026-06-09

Goal: make TokenGoblin fully production-ready end-to-end (backend, frontend, tests, CI, docs, Vercel deploy) with zero known compile/test regressions.

## Current verified baseline
- `go build ./...` passing
- `go test ./...` passing (18 test files)
- `npm run build` passing (Next.js 16.2.6)
- Frontend pages present: `/`, `/about`, `/pricing`, `/signup`, `/billing`, `/executive`, `/forecasts`, `/intelligence`, `/models`
- Frontend API routes present: `/api/billing/checkout`, `/api/billing/portal`, `/api/billing/status`, `/api/stripe/webhook`, `/api/tenant/register`

## Launch block 1: Vercel readiness
1. Confirm git-to-Vercel config requirement: project must be imported and env vars set in Vercel dashboard (Vercel does not use `vercel.json` alone for runtime secrets).
2. Add CI frontend install + build step already done in `.github/workflows/ci.yml`; verify workflow is enabled.
3. Optional but recommended: capture deploy preview URL once merged; if Vercel CLI is installed, add `vercel links this` step.

Done:
- `.github/workflows/ci.yml` now runs Go tests, lint, frontend build, frontend tests.

## Launch block 2: Frontend tests + coverage
1. Add Jest config working with Next.js types: keep `testEnvironment=jsdom`, use `@testing-library/react`.
2. Smoke-test the landing path page: feed success, error, empty API response shapes to critical routes.
3. Add minimum 2 test files covering critical flow:
   - `frontend/src/app/executive/__tests__/page.test.tsx` ✅ added
   - Add one more lanes for `/billing` or `/pricing` if time allows.

Next:
- Add `frontend/src/app/page.test.tsx`
- Add `frontend/src/app/billing/__tests__/page.test.tsx`

## Launch block 3: Backend safety hardening
1. Add backend integration smoke test around `/internal/billing/stripe-event` using httptest server.
2. Expand anomaly / intelligence coverage to ensure degraded path behavior is stable.
3. Add lint step in CI currently only runs Go tests; add `golangci-lint` if not already verifying in CI.

## Launch block 4: Observability + ops docs
1. Document required Vercel env vars and Stripe Price IDs in README with required/optional tags.
2. Document runtime ports and expected health endpoint behavior.
3. Add repo deprecation note for demo tenant mode in production startup path.

## Launch block 5: Final end-to-end verification checklist
- `go build ./...`
- `go test ./...`
- `npm run build`
- `npm test`
- CI workflow green on a throwaway branch
- Frontend header links include new routes
- Backend startup on `PORT=8080` with Postgres optional fallback to SQLite

## Immediate execution plan (ordered)
1. Finish frontend test suite with 2–3 test files (executive already done; add page + billing).
2. Patch any failing tests immediately.
3. Vercel deployment config verification steps.
4. README/documentation patch with launch-ready env instructions.
5. Run full verification pass and produce final status: ✅/❌ per item.
