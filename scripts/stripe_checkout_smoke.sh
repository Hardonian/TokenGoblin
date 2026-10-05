#!/usr/bin/env bash
# Stripe TEST-MODE checkout round trip for TokenGoblin (local or CI).
#
# Proves the money path end to end against real Stripe:
#   1. checkout session creation through the live Go API (allowlist -> Stripe)
#   2. signed webhook delivery (stripe listen -> our raw-body verifier)
#   3. billing lifecycle application (tenant tier verified via /api/billing/status)
#
# Usage:
#   ./scripts/stripe_checkout_smoke.sh --trigger-only   # signed test event (no browser)
#   ./scripts/stripe_checkout_smoke.sh                  # full: real session + waits for your test-card payment
#
# Environment (locally: .env, gitignored — NEVER commit):
#   STRIPE_SECRET_KEY=sk_test_...   (test mode ONLY; this script refuses sk_live_)
#   STRIPE_PRICE_PRO=price_...      (optional — an ephemeral test price is created if unset)
#   TG_INTERNAL_WEBHOOK_SECRET=...  (optional — generated if unset)
#   TG_DB_DSN=postgres://...        (optional — defaults to the compose Postgres)
# The webhook signing secret comes from the `stripe listen` session itself.
set -Eeuo pipefail

cd "$(dirname "$0")/.."

MODE="full"
[ "${1:-}" = "--trigger-only" ] && MODE="trigger"

# ---------- preflight ----------
if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

die() { echo "FAIL: $*" >&2; exit 1; }

[ -n "${STRIPE_SECRET_KEY:-}" ] || die "STRIPE_SECRET_KEY not set. Provide a TEST key (sk_test_...)."
case "$STRIPE_SECRET_KEY" in
  sk_live_*) die "STRIPE_SECRET_KEY is a LIVE key. This smoke test is test-mode only — refusing." ;;
  sk_test_*) : ;;
  *) die "STRIPE_SECRET_KEY does not look like a Stripe key (want sk_test_...)." ;;
esac

command -v stripe >/dev/null || die "stripe CLI not found. Install: https://stripe.com/docs/stripe-cli"
command -v curl >/dev/null || die "curl not found"
command -v go >/dev/null || die "go not found (needed to run the local API under test)"

# The Stripe CLI authenticates headless with the API key (no interactive login).
export STRIPE_API_KEY="$STRIPE_SECRET_KEY"

TG_DB_DSN="${TG_DB_DSN:-postgres://tokengoblin:tokengoblin-dev@127.0.0.1:55432/tokengoblin?sslmode=disable}"
TG_INTERNAL_WEBHOOK_SECRET="${TG_INTERNAL_WEBHOOK_SECRET:-$(openssl rand -hex 32)}"
API_PORT=8081
API_BASE="http://127.0.0.1:${API_PORT}"
TENANT="stripe-smoke-$(date +%s)"

echo "== Stripe checkout smoke (${MODE}) — tenant ${TENANT} =="
echo "-- step 0: stripe CLI reachable"
stripe --version

# ---------- price (ephemeral test price if none configured) ----------
if [ -z "${STRIPE_PRICE_PRO:-}" ]; then
  echo "-- step 0a: creating ephemeral test price (STRIPE_PRICE_PRO unset)"
  PRODUCT_JSON="$(curl -fsS -u "${STRIPE_SECRET_KEY}:" https://api.stripe.com/v1/products \
    --data-urlencode "name=TokenGoblin Smoke ${TENANT}")" || die "product creation failed"
  PRODUCT_ID="$(printf '%s' "$PRODUCT_JSON" | grep -o '"id": *"prod_[A-Za-z0-9]*"' | head -1 | grep -o 'prod_[A-Za-z0-9]*')"
  [ -n "$PRODUCT_ID" ] || die "no product id in: $PRODUCT_JSON"
  PRICE_JSON="$(curl -fsS -u "${STRIPE_SECRET_KEY}:" https://api.stripe.com/v1/prices \
    --data-urlencode "product=${PRODUCT_ID}" \
    --data-urlencode "unit_amount=2500" \
    --data-urlencode "currency=usd" \
    --data-urlencode "recurring[interval]=month")" || die "price creation failed"
  STRIPE_PRICE_PRO="$(printf '%s' "$PRICE_JSON" | grep -o '"id": *"price_[A-Za-z0-9]*"' | head -1 | grep -o 'price_[A-Za-z0-9]*')"
  [ -n "$STRIPE_PRICE_PRO" ] || die "no price id in: $PRICE_JSON"
  echo "   ephemeral price: ${STRIPE_PRICE_PRO}"
fi

# ---------- webhook listener (owns the signing secret for this run) ----------
LISTEN_LOG="$(mktemp /tmp/stripe_listen.XXXXXX.log)"
stripe listen --forward-to "${API_BASE}/api/v1/webhooks/stripe" >"$LISTEN_LOG" 2>&1 &
LISTEN_PID=$!
cleanup() {
  kill "$LISTEN_PID" 2>/dev/null || true
  if [ -n "${API_PID:-}" ]; then
    kill "$API_PID" 2>/dev/null || true
    wait "$API_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

WHSEC=""
for _ in $(seq 1 30); do
  WHSEC="$(grep -o 'whsec_[A-Za-z0-9]*' "$LISTEN_LOG" | head -1 || true)"
  [ -n "$WHSEC" ] && break
  kill -0 "$LISTEN_PID" 2>/dev/null || die "stripe listen exited early: $(cat "$LISTEN_LOG")"
  sleep 1
done
[ -n "$WHSEC" ] || die "no signing secret from stripe listen: $(cat "$LISTEN_LOG")"
echo "-- step 1: webhook listener ready (signing secret obtained from this session, not printed)"

# ---------- API under test (real binary, real DB, real raw-body verifier) ----------
# Build first, then run the BINARY directly: `go run` would spawn a child
# process that survives `kill $API_PID` and keeps CI steps hanging forever.
echo "-- step 2: starting API under test on :${API_PORT}"
go build -o /tmp/tg-stripe-smoke-server ./cmd/server
STRIPE_SECRET_KEY="$STRIPE_SECRET_KEY" \
STRIPE_WEBHOOK_SECRET="$WHSEC" \
STRIPE_PRICE_PRO="$STRIPE_PRICE_PRO" \
STRIPE_PRICE_ENTERPRISE="${STRIPE_PRICE_ENTERPRISE:-}" \
TG_INTERNAL_WEBHOOK_SECRET="$TG_INTERNAL_WEBHOOK_SECRET" \
TG_DB_DSN="$TG_DB_DSN" \
TG_ADDR=":${API_PORT}" \
/tmp/tg-stripe-smoke-server >/tmp/tg_stripe_smoke_api.log 2>&1 &
API_PID=$!

for _ in $(seq 1 60); do
  curl -fsS "${API_BASE}/healthz" >/dev/null 2>&1 && break
  kill -0 "$API_PID" 2>/dev/null || die "API exited early: $(tail -20 /tmp/tg_stripe_smoke_api.log)"
  sleep 1
done
curl -fsS "${API_BASE}/healthz" >/dev/null || die "API did not become healthy"
echo "   API healthy (raw-body webhook verifier active)"

# ---------- checkout session through the live code path ----------
echo "-- step 3: creating checkout session via POST /api/billing/checkout"
CHECKOUT="$(curl -fsS -X POST "${API_BASE}/api/billing/checkout" \
  -H "x-tenant-id: ${TENANT}" -H 'content-type: application/json' \
  -d "{\"price_id\":\"${STRIPE_PRICE_PRO}\",\"success_url\":\"https://example.com/success\",\"cancel_url\":\"https://example.com/cancel\"}")"
echo "$CHECKOUT" | grep -q '"ok":true' || die "checkout creation failed: $CHECKOUT"
SESSION_URL="$(echo "$CHECKOUT" | grep -o 'https://checkout.stripe.com[^"]*' | head -1 || true)"
[ -n "$SESSION_URL" ] || die "no checkout URL in response: $CHECKOUT"
echo "   session created: ${SESSION_URL}"

if [ "$MODE" = "full" ]; then
  echo
  echo "== COMPLETE THE PAYMENT NOW =="
  echo "   Open: ${SESSION_URL}"
  echo "   Test card: 4242 4242 4242 4242, any future expiry, any CVC, any ZIP."
  echo "   Waiting for the signed webhook (checkout.session.completed) ..."
  for _ in $(seq 1 240); do
    grep -q 'checkout.session.completed' "$LISTEN_LOG" && break
    sleep 2
  done
else
  echo "-- step 4: firing signed test event through the listener, tagged with our tenant"
  stripe trigger checkout.session.completed \
    --add "checkout_session.metadata.tenant_id=${TENANT}" \
    --add "checkout_session.client_reference_id=${TENANT}" \
    >/dev/null 2>&1 || die "stripe trigger failed (is the CLI authenticated?)"
  sleep 5
fi

# ---------- verify the lifecycle actually applied (through the product's own API) ----------
echo "-- step 5: verifying billing lifecycle via GET /api/billing/status"
STATUS="$(curl -fsS -H "x-tenant-id: ${TENANT}" "${API_BASE}/api/billing/status")"
echo "   status: ${STATUS}"
TIER="$(printf '%s' "$STATUS" | grep -o '"tier":"[a-z]*"' | head -1 | sed 's/.*:"//;s/"//')"
if [ -n "$TIER" ] && [ "$TIER" != "free" ]; then
  echo "PASS: billing lifecycle applied — tenant ${TENANT} moved to tier '${TIER}' via a real Stripe-signed webhook."
  exit 0
fi
echo "FAIL: webhook did not reach the billing lifecycle (tier '${TIER:-none}')." >&2
echo "   listener log: ${LISTEN_LOG}" >&2
echo "   api log: /tmp/tg_stripe_smoke_api.log" >&2
exit 1