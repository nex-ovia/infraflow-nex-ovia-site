#!/usr/bin/env bash
# One-time setup for nex-ovia.com on Cloudflare + GitHub Actions. Run on YOUR machine:
#
#   gh auth login            # if not already
#   bash scripts/bootstrap.sh
#
# Secrets are typed into hidden prompts and sent straight to GitHub (encrypted by gh). Nothing is
# written to disk or echoed. Safe to re-run: every step is idempotent.
# Needs: gh (logged in, admin on the repo), curl, node/npx (for wrangler), jq.
set -euo pipefail

REPO="${REPO:-nex-ovia/infraflow-nex-ovia-site}"
ACCOUNT_ID="${ACCOUNT_ID:-61d9d1af253451a04a3eb001a8af3227}"
ZONE_NAME="${ZONE_NAME:-nex-ovia.com}"
BUCKET="${BUCKET:-nexovia-tfstate}"
WRANGLER="wrangler@4.159.0"
CF_API="https://api.cloudflare.com/client/v4"

say()  { printf '\n\033[1;34m==>\033[0m %s\n' "$*"; }
ok()   { printf '    \033[32m✓\033[0m %s\n' "$*"; }
die()  { printf '\033[31merror:\033[0m %s\n' "$*" >&2; exit 1; }
ask_secret() { local v; read -r -s -p "    $1: " v; echo >&2; [[ -n "$v" ]] || die "$1 is required"; printf '%s' "$v"; }

for bin in gh curl jq npx; do command -v "$bin" >/dev/null || die "$bin is not installed"; done
gh auth status >/dev/null 2>&1 || die "run 'gh auth login' first"
gh repo view "$REPO" --json viewerPermission -q .viewerPermission | grep -q ADMIN || die "you need admin on $REPO"

say "1/6 Repository variable CF_ACCOUNT_ID"
gh variable set CF_ACCOUNT_ID --repo "$REPO" --body "$ACCOUNT_ID"
ok "CF_ACCOUNT_ID=$ACCOUNT_ID"

say "2/6 Cloudflare API token"
cat <<TXT
    Create it at https://dash.cloudflare.com/profile/api-tokens → Create Token → Custom token:
      Account · Cloudflare Pages · Edit         Zone · Zone Settings · Edit
      Account · Workers R2 Storage · Edit       Zone · DNS · Edit
      Account · Account Rulesets · Read         Zone · Zone WAF · Edit
      Account · Web Analytics · Edit            Zone · Single Redirect · Edit
      Zone · Zone · Read                        Zone · Bot Management · Edit
    Account resources: Include → "Nexovia - A futuristic path"
    Zone resources:    Include → Specific zone → $ZONE_NAME
    TTL: 1 year.
TXT
CF_TOKEN=$(ask_secret "Paste the Cloudflare API token (hidden)")
status=$(curl -fsS -H "Authorization: Bearer $CF_TOKEN" "$CF_API/user/tokens/verify" | jq -r .result.status) \
  || die "token verification failed"
[[ "$status" == "active" ]] || die "token status is '$status'"
ok "token is active"
ZONE_ID=$(curl -fsS -H "Authorization: Bearer $CF_TOKEN" "$CF_API/zones?name=$ZONE_NAME" | jq -r '.result[0].id // empty')
[[ -n "$ZONE_ID" ]] || die "token can't see zone $ZONE_NAME — add Zone · Zone · Read for it"
gh variable set CF_ZONE_ID --repo "$REPO" --body "$ZONE_ID"
ok "CF_ZONE_ID=$ZONE_ID"
printf '%s' "$CF_TOKEN" | gh secret set CLOUDFLARE_API_TOKEN --repo "$REPO"
ok "secret CLOUDFLARE_API_TOKEN set"

say "3/6 R2 bucket for Terraform state ($BUCKET)"
if CLOUDFLARE_API_TOKEN="$CF_TOKEN" CLOUDFLARE_ACCOUNT_ID="$ACCOUNT_ID" npx --yes "$WRANGLER" r2 bucket list 2>/dev/null | grep -Eq "name:[[:space:]]*$BUCKET([[:space:]]|$)"; then
  ok "bucket exists"
else
  CLOUDFLARE_API_TOKEN="$CF_TOKEN" CLOUDFLARE_ACCOUNT_ID="$ACCOUNT_ID" npx --yes "$WRANGLER" r2 bucket create "$BUCKET" >/dev/null \
    || die "could not create bucket. Enable R2 once in the dashboard (R2 → Get started; free tier, asks for a card) and re-run"
  ok "bucket created"
fi
unset CF_TOKEN

say "4/6 R2 access keys (bucket-scoped)"
cat <<TXT
    https://dash.cloudflare.com/$ACCOUNT_ID/r2/api-tokens → Create Account API token
      Permissions: Object Read & Write · Specify bucket: $BUCKET · TTL: Forever
    Copy the "Access Key ID" and "Secret Access Key" shown once.
TXT
printf '%s' "$(ask_secret "R2 Access Key ID (hidden)")" | gh secret set R2_ACCESS_KEY_ID --repo "$REPO"
printf '%s' "$(ask_secret "R2 Secret Access Key (hidden)")" | gh secret set R2_SECRET_ACCESS_KEY --repo "$REPO"
ok "R2 secrets set"

say "5/6 GitHub environments"
ME_ID=$(gh api user -q .id)
gh api -X PUT "repos/$REPO/environments/production" --input - >/dev/null <<JSON
{"reviewers":[{"type":"User","id":$ME_ID}],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}
JSON
gh api -X POST "repos/$REPO/environments/production/deployment-branch-policies" -f name=dev -f type=branch >/dev/null 2>&1 || true
ok "production (you approve every terraform apply; dev branch only)"
for env in site preview social; do
  gh api -X PUT "repos/$REPO/environments/$env" >/dev/null
  ok "$env"
done

say "6/6 First run"
gh workflow run infra.yml --repo "$REPO" --ref dev
ok "infra workflow started: gh run list --repo $REPO --workflow infra.yml"
cat <<TXT

Done. Next:
  • Approve the "apply" job when it asks (Actions → infra → Review deployments).
  • Then run the site workflow:  gh workflow run site.yml --repo $REPO --ref dev
  • Preview: https://nexovia-site.pages.dev
  • Social credentials (optional) go in the "social" environment — see docs/social.md.
TXT
