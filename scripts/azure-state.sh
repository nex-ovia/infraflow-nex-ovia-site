#!/usr/bin/env bash
# Create the Azure home for Terraform state and let GitHub Actions use it WITHOUT any stored secret
# (GitHub OIDC → Entra ID workload identity federation). Run once on your machine:
#
#   az login                 # account with Owner (or Contributor + User Access Administrator) on the subscription
#   gh auth login            # admin on the repo
#   bash scripts/azure-state.sh            # optional: LOCATION=centralindia SUBSCRIPTION=<id> bash ...
#
# Creates (idempotent):
#   resource group  nexovia-tfstate-rg
#   storage account nexoviatfstate<random>  Standard_LRS, TLS 1.2, HTTPS only, no public blobs,
#                   shared-key auth OFF (Entra ID only), blob versioning + 30-day soft delete
#   container       tfstate (private)
#   Entra app       nexovia-site-terraform, with federated credentials for this repo's
#                   dev branch, pull requests and the 'production' environment only
#   role            Storage Blob Data Contributor on the tfstate container only
# and sets GitHub repository variables AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID,
# AZURE_TFSTATE_RG, AZURE_TFSTATE_ACCOUNT. These are identifiers, not secrets.
# Cost: a few KB of blob storage, typically well under $0.01/month.
set -euo pipefail

REPO="${REPO:-nex-ovia/infraflow-nex-ovia-site}"
LOCATION="${LOCATION:-eastus}"
RG="${RG:-nexovia-tfstate-rg}"
CONTAINER="tfstate"
APP_NAME="${APP_NAME:-nexovia-site-terraform}"

say() { printf '\n\033[1;34m==>\033[0m %s\n' "$*"; }
ok()  { printf '    \033[32m✓\033[0m %s\n' "$*"; }
die() { printf '\033[31merror:\033[0m %s\n' "$*" >&2; exit 1; }

for bin in az gh jq; do command -v "$bin" >/dev/null || die "$bin is not installed (brew install azure-cli gh jq)"; done
az account show >/dev/null 2>&1 || die "run 'az login' first"
gh auth status >/dev/null 2>&1 || die "run 'gh auth login' first"
[[ -n "${SUBSCRIPTION:-}" ]] && az account set --subscription "$SUBSCRIPTION"
SUB_ID=$(az account show --query id -o tsv)
TENANT_ID=$(az account show --query tenantId -o tsv)
ok "subscription $(az account show --query name -o tsv) ($SUB_ID)"

say "1/5 Resource group and storage account"
az provider register --namespace Microsoft.Storage --wait >/dev/null
az group create -n "$RG" -l "$LOCATION" --tags purpose=terraform-state project=nex-ovia.com -o none
# Reuse the account recorded in GitHub (or an existing one in the group) so re-runs don't create another.
SA=$(gh variable get AZURE_TFSTATE_ACCOUNT --repo "$REPO" 2>/dev/null || true)
[[ -n "$SA" ]] || SA=$(az storage account list -g "$RG" --query "[0].name" -o tsv)
[[ -n "$SA" ]] || SA="nexoviatfstate$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 8)"
if ! az storage account show -n "$SA" -g "$RG" -o none 2>/dev/null; then
  az storage account create -n "$SA" -g "$RG" -l "$LOCATION" --sku Standard_LRS --kind StorageV2 \
    --min-tls-version TLS1_2 --https-only true --allow-blob-public-access false \
    --allow-shared-key-access false --default-to-oauth-authentication true -o none
fi
az storage account blob-service-properties update --account-name "$SA" -g "$RG" \
  --enable-versioning true --enable-delete-retention true --delete-retention-days 30 \
  --enable-container-delete-retention true --container-delete-retention-days 30 -o none
ok "storage account $SA (versioning + 30-day soft delete, Entra-only auth)"

say "2/5 Private container '$CONTAINER'"
# ARM (control-plane) call: works even though shared-key access is off and before data roles exist.
az storage container-rm create --storage-account "$SA" -g "$RG" -n "$CONTAINER" --public-access off -o none 2>/dev/null \
  || az storage container-rm show --storage-account "$SA" -g "$RG" -n "$CONTAINER" -o none
ok "container $CONTAINER"

say "3/5 Entra app + federated credentials (GitHub OIDC, no secret)"
APP_ID=$(az ad app list --display-name "$APP_NAME" --query "[0].appId" -o tsv)
[[ -n "$APP_ID" ]] || APP_ID=$(az ad app create --display-name "$APP_NAME" --sign-in-audience AzureADMyOrg --query appId -o tsv)
SP_ID=$(az ad sp show --id "$APP_ID" --query id -o tsv 2>/dev/null || az ad sp create --id "$APP_ID" --query id -o tsv)
existing=$(az ad app federated-credential list --id "$APP_ID" --query "[].name" -o tsv)
add_fic() {  # name subject
  grep -qx "$1" <<<"$existing" && { ok "federated credential $1 (exists)"; return; }
  az ad app federated-credential create --id "$APP_ID" -o none --parameters "$(jq -n --arg n "$1" --arg s "$2" \
    '{name:$n, issuer:"https://token.actions.githubusercontent.com", subject:$s, audiences:["api://AzureADTokenExchange"]}')"
  ok "federated credential $1 → $2"
}
add_fic gh-dev-branch   "repo:$REPO:ref:refs/heads/dev"
add_fic gh-pull-request "repo:$REPO:pull_request"
add_fic gh-production   "repo:$REPO:environment:production"

say "4/5 Least-privilege role on the container only"
SCOPE="/subscriptions/$SUB_ID/resourceGroups/$RG/providers/Microsoft.Storage/storageAccounts/$SA/blobServices/default/containers/$CONTAINER"
if [[ -z "$(az role assignment list --assignee "$SP_ID" --scope "$SCOPE" --role "Storage Blob Data Contributor" --query "[0].id" -o tsv)" ]]; then
  az role assignment create --assignee-object-id "$SP_ID" --assignee-principal-type ServicePrincipal \
    --role "Storage Blob Data Contributor" --scope "$SCOPE" -o none
fi
ok "Storage Blob Data Contributor on $CONTAINER"

say "5/5 GitHub repository variables (identifiers, not secrets)"
for kv in "AZURE_CLIENT_ID=$APP_ID" "AZURE_TENANT_ID=$TENANT_ID" "AZURE_SUBSCRIPTION_ID=$SUB_ID" \
          "AZURE_TFSTATE_RG=$RG" "AZURE_TFSTATE_ACCOUNT=$SA"; do
  gh variable set "${kv%%=*}" --repo "$REPO" --body "${kv#*=}"
  ok "${kv%%=*}"
done

cat <<TXT

Done. Role assignments can take a minute or two to apply. Then re-run the infra workflow:
  gh run list --repo $REPO --workflow infra.yml --limit 1
  gh run rerun <run-id> --repo $REPO --failed
R2 is no longer used; you can ignore the earlier R2 step.
TXT
