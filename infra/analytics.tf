# Cookieless Cloudflare Web Analytics (free). auto_install injects the beacon on proxied traffic.
resource "cloudflare_web_analytics_site" "site" {
  account_id   = var.account_id
  zone_tag     = var.zone_id
  auto_install = true
}
