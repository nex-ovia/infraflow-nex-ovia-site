#!/usr/bin/env bash
# Plan the Cloudflare stack: adopt existing objects (cfimport), plan, then fail if any protected
# resource would be deleted (planguard). Writes infra/tfplan and infra/plan-summary.txt.
# Needs: CLOUDFLARE_API_TOKEN, TF_VAR_account_id (zone ID is derived in cf-env.sh), and for the Azure
# state backend: ARM_CLIENT_ID, ARM_TENANT_ID, ARM_SUBSCRIPTION_ID (OIDC), TFSTATE_RG, TFSTATE_ACCOUNT.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=infra/scripts/cf-env.sh
. scripts/cf-env.sh

cutover=$(sed -n 's/^cutover *= *\(true\|false\).*/\1/p' env/prod.tfvars)
flags=()
allow=""
if [[ "$cutover" == "true" ]]; then
  flags+=(-cutover)
  # www/apex change record type at cutover, so a replace is expected for exactly these two.
  allow="cloudflare_dns_record.www[0],cloudflare_dns_record.apex[0]"
fi

(cd ../tools && go run ./cfimport -account "$TF_VAR_account_id" -zone "$TF_VAR_zone_id" "${flags[@]}") > import.tf

: "${TFSTATE_RG:?set repository variable AZURE_TFSTATE_RG (run scripts/azure-state.sh)}"
: "${TFSTATE_ACCOUNT:?set repository variable AZURE_TFSTATE_ACCOUNT (run scripts/azure-state.sh)}"
terraform init -input=false \
  -backend-config="resource_group_name=$TFSTATE_RG" \
  -backend-config="storage_account_name=$TFSTATE_ACCOUNT"
terraform plan -input=false -lock-timeout=5m -var-file=env/prod.tfvars -out=tfplan -no-color | tee plan-full.txt >/dev/null
terraform show -json tfplan | (cd ../tools && go run ./planguard -allow "$allow")
grep -E '^(Plan:|No changes\.|  # )' plan-full.txt > plan-summary.txt || true
cat plan-summary.txt
