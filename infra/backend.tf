# Remote state on Cloudflare R2 (S3-compatible, free tier). Bucket and keys are created by hand (chunk U2);
# credentials come from AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars in CI.
# The endpoint (https://<account_id>.r2.cloudflarestorage.com) is passed at init:
#   terraform init -backend-config="endpoints={s3=\"https://<account_id>.r2.cloudflarestorage.com\"}"
terraform {
  backend "s3" {
    bucket = "nexovia-tfstate"
    key    = "nex-ovia.com/prod.tfstate"
    region = "auto"

    use_lockfile                = true
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    skip_s3_checksum            = true
    use_path_style              = true
  }
}
