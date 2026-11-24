locals {
  site_hosts_expr = "http.host in {\"${var.site_host}\" \"${var.apex}\"}"
}

resource "cloudflare_ruleset" "redirects" {
  zone_id     = var.zone_id
  name        = "nexovia redirects"
  description = "apex -> www, legacy paths"
  kind        = "zone"
  phase       = "http_request_dynamic_redirect"
  rules = [
    {
      ref         = "apex_to_www"
      description = "Redirect apex to canonical www host"
      expression  = "http.host eq \"${var.apex}\""
      action      = "redirect"
      action_parameters = {
        from_value = {
          status_code           = 301
          preserve_query_string = true
          target_url = {
            expression = "concat(\"https://${var.site_host}\", http.request.uri.path)"
          }
        }
      }
    },
    {
      ref         = "sponser_typo"
      description = "Old misspelled sponsor path"
      expression  = "${local.site_hosts_expr} and starts_with(http.request.uri.path, \"/sponser\")"
      action      = "redirect"
      action_parameters = {
        from_value = {
          status_code           = 301
          preserve_query_string = true
          target_url = {
            value = "https://${var.site_host}/sponsor/"
          }
        }
      }
    },
  ]
}

resource "cloudflare_ruleset" "firewall" {
  zone_id     = var.zone_id
  name        = "nexovia firewall"
  description = "Static site: only GET/HEAD/OPTIONS"
  kind        = "zone"
  phase       = "http_request_firewall_custom"
  rules = [
    {
      ref         = "static_methods_only"
      description = "Block methods a static site never serves"
      expression  = "${local.site_hosts_expr} and not http.request.method in {\"GET\" \"HEAD\" \"OPTIONS\"}"
      action      = "block"
    },
  ]
}

resource "cloudflare_ruleset" "ratelimit" {
  zone_id     = var.zone_id
  name        = "nexovia rate limit"
  description = "Per-IP burst limit (free plan: one rule, 10s window)"
  kind        = "zone"
  phase       = "http_ratelimit"
  rules = [
    {
      ref         = "burst_per_ip"
      description = "Block an IP that exceeds the burst limit"
      expression  = local.site_hosts_expr
      action      = "block"
      ratelimit = {
        characteristics     = ["cf.colo.id", "ip.src"]
        period              = 10
        requests_per_period = var.rate_limit_requests
        mitigation_timeout  = 10
      }
    },
  ]
}
