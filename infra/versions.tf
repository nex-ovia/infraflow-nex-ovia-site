terraform {
  required_version = ">= 1.10.0"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.28"
    }
  }
}

# Token comes from the CLOUDFLARE_API_TOKEN environment variable (GitHub secret); never from files.
provider "cloudflare" {}
