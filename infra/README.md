# infra: nex-ovia.com on Cloudflare (Terraform)

| File | Manages |
|---|---|
| `pages.tf` | Pages project `nexovia-site` (direct upload) and, at cutover, the `www` custom domain |
| `dns.tf` | `www`/apex records (cutover only) and CAA records. Every other record in the zone is left alone |
| `security.tf` | SSL strict, TLS ≥ 1.2, TLS 1.3, Always HTTPS, HSTS, DNSSEC |
| `rules.tf` | apex → www and `/sponser` redirects, static-methods-only firewall rule, per-IP rate limit |
| `bots.tf` | AI crawler policy (Cloudflare-managed robots.txt and AI-bot blocking off) |
| `analytics.tf` | Cookieless Web Analytics |

- **Run:** CI only (`.github/workflows/infra.yml`). Each PR gets checks and a guarded plan; `dev` applies after approval in the `production` environment.
- **Local check:** `terraform init -backend=false && terraform validate && terraform test`.
- **Safety:** `tools/cfimport` adopts existing objects, and `tools/planguard` fails any plan that deletes DNS records or the Pages project.
- **Cutover:** set `cutover = true` in `env/prod.tfvars`. See `../docs/setup.md`.
