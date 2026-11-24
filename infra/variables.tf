variable "account_id" {
  description = "Cloudflare account ID (not secret)."
  type        = string
  validation {
    condition     = can(regex("^[0-9a-f]{32}$", var.account_id))
    error_message = "account_id must be a 32-char hex Cloudflare account ID."
  }
}

variable "zone_id" {
  description = "Cloudflare zone ID for nex-ovia.com (not secret)."
  type        = string
  validation {
    condition     = can(regex("^[0-9a-f]{32}$", var.zone_id))
    error_message = "zone_id must be a 32-char hex Cloudflare zone ID."
  }
}

variable "apex" {
  description = "Apex domain."
  type        = string
  default     = "nex-ovia.com"
}

variable "site_host" {
  description = "Canonical hostname the site is served from."
  type        = string
  default     = "www.nex-ovia.com"
}

variable "pages_project" {
  description = "Cloudflare Pages project name (also the *.pages.dev preview subdomain)."
  type        = string
  default     = "nexovia-site"
  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{0,56}[a-z0-9]$", var.pages_project))
    error_message = "pages_project must be lowercase letters, digits and hyphens."
  }
}

variable "cutover" {
  description = "false: build everything beside the live Azure site. true: attach www/apex to Pages and repoint DNS (chunk C02)."
  type        = bool
  default     = false
}

variable "enable_dnssec" {
  description = "Sign the zone. Harmless until the DS record is added at the registrar."
  type        = bool
  default     = true
}

variable "caa_issuers" {
  description = "CAs allowed to issue certs. Cloudflare Universal SSL CAs + Let's Encrypt (GitHub Pages subdomains) + DigiCert (Azure SWA during rollback window)."
  type        = list(string)
  default     = ["letsencrypt.org", "pki.goog", "ssl.com", "digicert.com"]
  validation {
    condition     = length(var.caa_issuers) > 0 && alltrue([for c in var.caa_issuers : can(regex("^[a-z0-9.-]+$", c))])
    error_message = "caa_issuers must be non-empty bare domain names."
  }
}

variable "caa_report_email" {
  description = "Mailbox for CAA iodef violation reports."
  type        = string
  default     = "support@nex-ovia.com"
}

variable "rate_limit_requests" {
  description = "Requests per 10s per IP (per Cloudflare colo) before blocking. Free plan allows one rate-limit rule."
  type        = number
  default     = 150
  validation {
    condition     = var.rate_limit_requests >= 50
    error_message = "Keep the limit >= 50 so crawlers and normal browsing are never blocked."
  }
}

variable "ai_crawlers" {
  description = "AI crawler policy. allow_all: search, assistant and training bots may crawl. allow_search: only search/assistant bots."
  type        = string
  default     = "allow_all"
  validation {
    condition     = contains(["allow_all", "allow_search"], var.ai_crawlers)
    error_message = "ai_crawlers must be allow_all or allow_search."
  }
}
