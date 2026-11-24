# Offline tests: mocked provider, plan-only. Run: terraform test
mock_provider "cloudflare" {}

variables {
  account_id = "0123456789abcdef0123456789abcdef"
  zone_id    = "fedcba9876543210fedcba9876543210"
}

run "pre_cutover_leaves_live_site_alone" {
  command = plan
  assert {
    condition     = length(cloudflare_dns_record.www) == 0 && length(cloudflare_dns_record.apex) == 0
    error_message = "www/apex DNS must not be managed before cutover"
  }
  assert {
    condition     = length(cloudflare_pages_domain.www) == 0
    error_message = "custom domain must not attach before cutover"
  }
}

run "cutover_points_www_to_pages" {
  command = plan
  variables {
    cutover = true
  }
  assert {
    condition     = cloudflare_dns_record.www[0].content == "nexovia-site.pages.dev" && cloudflare_dns_record.www[0].proxied
    error_message = "www must be a proxied CNAME to the Pages project"
  }
  assert {
    condition     = cloudflare_dns_record.apex[0].proxied
    error_message = "apex must be proxied so the redirect rule applies"
  }
}

run "tls_and_hsts_are_strict" {
  command = plan
  assert {
    condition     = cloudflare_zone_setting.this["ssl"].value == "strict" && cloudflare_zone_setting.this["min_tls_version"].value == "1.2"
    error_message = "SSL must be strict with TLS >= 1.2"
  }
  assert {
    condition     = cloudflare_zone_setting.this["always_use_https"].value == "on"
    error_message = "Always Use HTTPS must be on"
  }
  assert {
    condition     = cloudflare_zone_setting.hsts.value.strict_transport_security.enabled && cloudflare_zone_setting.hsts.value.strict_transport_security.max_age >= 31536000
    error_message = "HSTS must be on with max-age >= 1 year"
  }
}

run "caa_restricts_issuers" {
  command = plan
  assert {
    condition     = length(cloudflare_dns_record.caa_issue) == 4 && contains(keys(cloudflare_dns_record.caa_issue), "pki.goog")
    error_message = "CAA must list exactly the configured issuers"
  }
}

run "redirect_and_firewall_rules" {
  command = plan
  assert {
    condition     = cloudflare_ruleset.redirects.rules[0].action_parameters.from_value.status_code == 301
    error_message = "apex->www must be a 301"
  }
  assert {
    condition     = strcontains(cloudflare_ruleset.firewall.rules[0].expression, "not http.request.method in")
    error_message = "firewall must restrict methods"
  }
  assert {
    condition     = cloudflare_ruleset.ratelimit.rules[0].ratelimit.period == 10
    error_message = "rate limit must use the free-plan 10s window"
  }
}

run "ai_crawlers_allowed" {
  command = plan
  assert {
    condition     = !cloudflare_bot_management.this.is_robots_txt_managed && cloudflare_bot_management.this.ai_bots_protection == "disabled"
    error_message = "Cloudflare must not manage robots.txt or block AI bots"
  }
  assert {
    condition     = cloudflare_bot_management.this.ai_training == "disabled"
    error_message = "allow_all must not restrict training bots"
  }
}

run "allow_search_restricts_training_only" {
  command = plan
  variables {
    ai_crawlers = "allow_search"
  }
  assert {
    condition     = cloudflare_bot_management.this.ai_training == "disallow" && cloudflare_bot_management.this.aisearch == "disabled"
    error_message = "allow_search must disallow training but keep search open"
  }
}

run "rejects_low_rate_limit" {
  command = plan
  variables {
    rate_limit_requests = 10
  }
  expect_failures = [var.rate_limit_requests]
}
