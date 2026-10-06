#!/usr/bin/env bash
# cf_deploy_preflight.sh — fail fast, loudly, and HONESTLY on bad Cloudflare creds.
#
# Contract:
#   exit 0 = CLOUDFLARE_API_TOKEN + CLOUDFLARE_ACCOUNT_ID pass shape checks AND
#            the token verifies live against the Cloudflare API.
#   exit 1 = configured but bad. Prints non-secret diagnostics only
#            (3-char family prefix + length), never a secret value.
#
# Why: a bad CF_API_TOKEN used to surface as wrangler noise (9109/10000 after
# retry cycles). Root-cause the credential here instead — same pattern that
# fixed the Stripe smoke preflight (scripts/stripe_checkout_smoke.sh).
#
# Usage: CLOUDFLARE_API_TOKEN=... CLOUDFLARE_ACCOUNT_ID=... ./scripts/cf_deploy_preflight.sh
#        (in CI both come from repo secrets CF_API_TOKEN / CF_ACCOUNT_ID)
set -Eeuo pipefail

die() { echo "PREFLIGHT FAIL: $*" >&2; exit 1; }

# ---------- presence ----------
[ -n "${CLOUDFLARE_API_TOKEN:-}" ]  || die "CLOUDFLARE_API_TOKEN is empty (repo secret CF_API_TOKEN not configured)."
[ -n "${CLOUDFLARE_ACCOUNT_ID:-}" ] || die "CLOUDFLARE_ACCOUNT_ID is empty (repo secret CF_ACCOUNT_ID not configured)."

# ---------- normalize (pasted secrets carry trailing newlines/spaces) ----------
CLOUDFLARE_API_TOKEN="$(printf '%s' "$CLOUDFLARE_API_TOKEN" | tr -d '[:space:]')"
CLOUDFLARE_ACCOUNT_ID="$(printf '%s' "$CLOUDFLARE_ACCOUNT_ID" | tr -d '[:space:]')"

shape_hint() { # $1 = value, $2 = label — 3-char family prefix + length only.
  echo "   ${2} shape: prefix='${1:0:3}...' length=${#1}" >&2
}

# ---------- shape (non-secret) ----------
case "$CLOUDFLARE_API_TOKEN" in
  *[!A-Za-z0-9_-]*) shape_hint "$CLOUDFLARE_API_TOKEN" "token"
    die "CLOUDFLARE_API_TOKEN contains characters outside [A-Za-z0-9_-] — likely a mangled paste." ;;
esac
if [ "${#CLOUDFLARE_API_TOKEN}" -lt 30 ] || [ "${#CLOUDFLARE_API_TOKEN}" -gt 200 ]; then
  shape_hint "$CLOUDFLARE_API_TOKEN" "token"
  die "CLOUDFLARE_API_TOKEN length out of range (want 30..200; real tokens are ~40) — truncated or placeholder value."
fi
case "$CLOUDFLARE_ACCOUNT_ID" in
  *[!0-9a-fA-F]*|'') shape_hint "$CLOUDFLARE_ACCOUNT_ID" "account id"
    die "CLOUDFLARE_ACCOUNT_ID is not 32-char hex." ;;
esac
if [ "${#CLOUDFLARE_ACCOUNT_ID}" -ne 32 ]; then
  shape_hint "$CLOUDFLARE_ACCOUNT_ID" "account id"
  die "CLOUDFLARE_ACCOUNT_ID length != 32 (got ${#CLOUDFLARE_ACCOUNT_ID})."
fi

# ---------- live verification (proves the value, not just the shape) ----------
echo "-- verifying token against https://api.cloudflare.com/client/v4/user/tokens/verify"
VERIFY_JSON="$(curl -sS --max-time 15 -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  https://api.cloudflare.com/client/v4/user/tokens/verify)" \
  || die "could not reach the Cloudflare API to verify the token (network/DNS); refusing to deploy blind."

if ! grep -q '"success":true' <<<"$VERIFY_JSON"; then
  shape_hint "$CLOUDFLARE_API_TOKEN" "token"
  # The API error body is non-secret (codes/messages only).
  die "Cloudflare rejected CLOUDFLARE_API_TOKEN (verify said: $(grep -o '"code":[0-9]*' <<<"$VERIFY_JSON" | head -1), $(grep -o '"message":"[^"]*"' <<<"$VERIFY_JSON" | head -1)). Rotate repo secret CF_API_TOKEN."
fi
if ! grep -q '"status":"active"' <<<"$VERIFY_JSON"; then
  shape_hint "$CLOUDFLARE_API_TOKEN" "token"
  die "CLOUDFLARE_API_TOKEN verifies but is not ACTIVE (disabled/expired). Rotate repo secret CF_API_TOKEN."
fi

echo "PREFLIGHT OK: token active, account id shape valid."
