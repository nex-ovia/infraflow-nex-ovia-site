# Direct-upload Pages project: content is pushed by wrangler from CI (site.yml), not built by Cloudflare.
resource "cloudflare_pages_project" "site" {
  account_id        = var.account_id
  name              = var.pages_project
  production_branch = "main"
}

# Custom domain is attached only at cutover so the live Azure site is untouched until then.
resource "cloudflare_pages_domain" "www" {
  count        = var.cutover ? 1 : 0
  account_id   = var.account_id
  project_name = cloudflare_pages_project.site.name
  name         = var.site_host
}

output "pages_dev_url" {
  description = "Preview hostname to verify before cutover."
  value       = "https://${var.pages_project}.pages.dev"
}
