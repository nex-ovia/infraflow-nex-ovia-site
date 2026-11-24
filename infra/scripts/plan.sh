#!/usr/bin/env bash
# Plan the Cloudflare stack: adopt existing objects (cfimport), plan, then fail if any protected
# resource would be deleted (planguard). Writes infra/tfplan and infra/plan-summary.txt.
# Needs: CLOUDFLARE_API_TOKEN and TF_VAR_account_id. Zone ID and R2 state credentials are derived (cf-env.sh).
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

terraform init -input=false \
  -backend-config="endpoints={s3=\"https://${TF_VAR_account_id}.r2.cloudflarestorage.com\"}"
terraform plan -input=false -lock-timeout=5m -var-file=env/prod.tfvars -out=tfplan -no-color | tee plan-full.txt >/dev/null
terraform show -json tfplan | (cd ../tools && go run ./planguard -allow "$allow")
grep -E '^(Plan:|No changes\.|  # )' plan-full.txt > plan-summary.txt || true
cat plan-summary.txt
