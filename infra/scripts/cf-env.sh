#!/usr/bin/env bash
# Derive everything Terraform needs from CLOUDFLARE_API_TOKEN + TF_VAR_account_id, so the repo only
# stores one secret and one variable:
#   - TF_VAR_zone_id      looked up by zone name (unless already set)
#   - R2 S3 credentials   Access Key ID = token id, Secret = sha256(token)  (Cloudflare R2 docs, "Get S3
#                         API credentials from an API token"); used only if R2_* secrets aren't set
#   - state bucket        created if missing (idempotent)
# Source it:  . infra/scripts/cf-env.sh
set -euo pipefail

: "${CLOUDFLARE_API_TOKEN:?set repository secret CLOUDFLARE_API_TOKEN}"
: "${TF_VAR_account_id:?set repository variable CF_ACCOUNT_ID}"
ZONE_NAME="${ZONE_NAME:-nex-ovia.com}"
STATE_BUCKET="${STATE_BUCKET:-nexovia-tfstate}"
cf() { curl -fsS -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" "$@"; }
api="https://api.cloudflare.com/client/v4"

# Account-owned tokens verify under the account; user tokens under /user.
token_id=$(cf "$api/accounts/$TF_VAR_account_id/tokens/verify" 2>/dev/null | jq -r '.result.id // empty' || true)
[[ -n "$token_id" ]] || token_id=$(cf "$api/user/tokens/verify" | jq -r '.result.id // empty')
[[ -n "$token_id" ]] || { echo "Cloudflare token could not be verified" >&2; exit 1; }

if [[ -z "${TF_VAR_zone_id:-}" ]]; then
  TF_VAR_zone_id=$(cf "$api/zones?name=$ZONE_NAME&account.id=$TF_VAR_account_id" | jq -r '.result[0].id // empty')
  [[ -n "$TF_VAR_zone_id" ]] || { echo "token can't read zone $ZONE_NAME (needs Zone · Zone · Read)" >&2; exit 1; }
fi
export TF_VAR_zone_id

if [[ -z "${AWS_ACCESS_KEY_ID:-}" || -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
  AWS_ACCESS_KEY_ID="$token_id"
  AWS_SECRET_ACCESS_KEY=$(printf '%s' "$CLOUDFLARE_API_TOKEN" | sha256sum | cut -d' ' -f1)
  [[ -n "${GITHUB_ACTIONS:-}" ]] && echo "::add-mask::$AWS_SECRET_ACCESS_KEY"
fi
export AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY

if ! cf "$api/accounts/$TF_VAR_account_id/r2/buckets/$STATE_BUCKET" >/dev/null 2>&1; then
  cf -X POST -H "Content-Type: application/json" "$api/accounts/$TF_VAR_account_id/r2/buckets" \
    --data "{\"name\":\"$STATE_BUCKET\"}" >/dev/null \
    || { echo "could not create R2 bucket $STATE_BUCKET: enable R2 once in the dashboard and give the token Workers R2 Storage · Edit" >&2; exit 1; }
  echo "created R2 bucket $STATE_BUCKET"
fi
echo "cloudflare env ready (zone ${TF_VAR_zone_id:0:6}…, state bucket $STATE_BUCKET)"
