#!/usr/bin/env bash
# One-time setup for nex-ovia.com on Cloudflare + GitHub Actions. Run on YOUR machine:
#
#   gh auth login            # if not already
#   bash scripts/bootstrap.sh
#
# You need exactly two things: the account ID (pre-filled) and one Cloudflare API token.
# Secrets are typed into hidden prompts and sent straight to GitHub (encrypted by gh). Nothing is
# written to disk or echoed. Safe to re-run: every step is idempotent.
# Needs: gh (logged in, admin on the repo), curl, jq.
set -euo pipefail

REPO="${REPO:-nex-ovia/infraflow-nex-ovia-site}"
ACCOUNT_ID="${ACCOUNT_ID:-61d9d1af253451a04a3eb001a8af3227}"
ZONE_NAME="${ZONE_NAME:-nex-ovia.com}"
CF_API="https://api.cloudflare.com/client/v4"

say()  { printf '\n\033[1;34m==>\033[0m %s\n' "$*"; }
ok()   { printf '    \033[32m✓\033[0m %s\n' "$*"; }
die()  { printf '\033[31merror:\033[0m %s\n' "$*" >&2; exit 1; }
ask_secret() { local v; read -r -s -p "    $1: " v; echo >&2; [[ -n "$v" ]] || die "$1 is required"; printf '%s' "$v"; }

for bin in gh curl jq; do command -v "$bin" >/dev/null || die "$bin is not installed"; done
gh auth status >/dev/null 2>&1 || die "run 'gh auth login' first"
gh repo view "$REPO" --json viewerPermission -q .viewerPermission | grep -q ADMIN || die "you need admin on $REPO"

say "1/4 Repository variable CF_ACCOUNT_ID"
gh variable set CF_ACCOUNT_ID --repo "$REPO" --body "$ACCOUNT_ID"
ok "CF_ACCOUNT_ID=$ACCOUNT_ID"

say "2/4 Cloudflare API token"
cat <<TXT
    Create it at https://dash.cloudflare.com/profile/api-tokens → Create Token → Custom token:
      Account · Cloudflare Pages · Edit         Zone · Zone Settings · Edit
      Zone · DNS · Edit
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

# Terraform state lives in Azure Storage: run scripts/azure-state.sh (no secret; GitHub OIDC).
unset CF_TOKEN

say "3/4 GitHub environments"
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

say "4/4 First run"
gh workflow run infra.yml --repo "$REPO" --ref dev
ok "infra workflow started: gh run list --repo $REPO --workflow infra.yml"
cat <<TXT

Done. Next:
  • Approve the "apply" job when it asks (Actions → infra → Review deployments).
  • Then run the site workflow:  gh workflow run site.yml --repo $REPO --ref dev
  • Preview: https://nexovia-site.pages.dev
  • Social credentials (optional) go in the "social" environment — see docs/social.md.
TXT
