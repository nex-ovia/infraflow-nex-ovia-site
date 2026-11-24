locals {
  zone_settings = {
    ssl                      = "strict"
    min_tls_version          = "1.2"
    tls_1_3                  = "on"
    always_use_https         = "on"
    automatic_https_rewrites = "on"
    opportunistic_encryption = "on"
    browser_check            = "on"
    brotli                   = "on"
    early_hints              = "on"
    http3                    = "on"
  }
}

resource "cloudflare_zone_setting" "this" {
  for_each   = local.zone_settings
  zone_id    = var.zone_id
  setting_id = each.key
  value      = each.value
}

# HSTS. preload stays false until the site has run cleanly on HTTPS for 30 days.
resource "cloudflare_zone_setting" "hsts" {
  zone_id    = var.zone_id
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled            = true
      max_age            = 31536000
      include_subdomains = true
      preload            = false
      nosniff            = true
    }
  }
}

resource "cloudflare_zone_dnssec" "this" {
  count   = var.enable_dnssec ? 1 : 0
  zone_id = var.zone_id
  status  = "active"
}
