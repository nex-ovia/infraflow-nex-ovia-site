# The site ships its own robots.txt (tools/nxsite). Cloudflare's managed robots.txt and AI-bot blocking
# are turned off so AI search, assistant and (optionally) training crawlers can read the site.
resource "cloudflare_bot_management" "this" {
  zone_id               = var.zone_id
  fight_mode            = false
  is_robots_txt_managed = false
  ai_bots_protection    = "disabled"
  crawler_protection    = "disabled"
  aisearch              = "disabled"
  ai_user               = "disabled"
  ai_training           = var.ai_crawlers == "allow_all" ? "disabled" : "disallow"
}
