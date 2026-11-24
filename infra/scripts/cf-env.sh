#!/usr/bin/env bash
# Verify the Cloudflare token and derive the zone ID from it, so the repo only stores
# CLOUDFLARE_API_TOKEN (secret) and CF_ACCOUNT_ID (variable). Source it:  . infra/scripts/cf-env.sh
set -euo pipefail

: "${CLOUDFLARE_API_TOKEN:?set repository secret CLOUDFLARE_API_TOKEN}"
: "${TF_VAR_account_id:?set repository variable CF_ACCOUNT_ID}"
ZONE_NAME="${ZONE_NAME:-nex-ovia.com}"
api="https://api.cloudflare.com/client/v4"
cf() { curl -fsS -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" "$@"; }

# Account-owned tokens verify under the account; user tokens under /user.
status=$(cf "$api/accounts/$TF_VAR_account_id/tokens/verify" 2>/dev/null | jq -r '.result.status // empty' || true)
[[ -n "$status" ]] || status=$(cf "$api/user/tokens/verify" | jq -r '.result.status // empty')
[[ "$status" == "active" ]] || { echo "::error::Cloudflare token is not active (status: ${status:-unknown})" >&2; exit 1; }

if [[ -z "${TF_VAR_zone_id:-}" ]]; then
  TF_VAR_zone_id=$(cf "$api/zones?name=$ZONE_NAME&account.id=$TF_VAR_account_id" | jq -r '.result[0].id // empty')
  [[ -n "$TF_VAR_zone_id" ]] || { echo "::error::token can't read zone $ZONE_NAME (needs Zone · Zone · Read)" >&2; exit 1; }
fi
export TF_VAR_zone_id
echo "cloudflare env ready (zone ${TF_VAR_zone_id:0:6}…)"
