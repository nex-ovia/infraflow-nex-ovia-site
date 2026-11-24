# Remote state in Azure Blob Storage (container "tfstate"). Locking uses blob leases.
# CI signs in with GitHub OIDC → Entra ID workload identity federation: no stored Azure secret.
# Storage account / resource group come from GitHub variables at init (infra/scripts/plan.sh);
# ARM_CLIENT_ID, ARM_TENANT_ID, ARM_SUBSCRIPTION_ID are set by the workflow. Created by scripts/azure-state.sh.
terraform {
  backend "azurerm" {
    container_name   = "tfstate"
    key              = "nex-ovia.com/prod.tfstate"
    use_oidc         = true
    use_azuread_auth = true
  }
}
