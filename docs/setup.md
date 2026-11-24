# One-time setup (free tiers only)

**Fast path:** on your own machine, with `gh` logged in:

```bash
git clone https://github.com/nex-ovia/infraflow-nex-ovia-site && cd infraflow-nex-ovia-site && git checkout dev
bash scripts/bootstrap.sh
```

The script does steps 1–5 below. It prompts (hidden) for the two things only you can create, the Cloudflare API token and the R2 keys, and starts the first run. The manual steps are kept below for reference.

Everything else is automated. Do these once, in order. Nothing here costs money.

## 1. Cloudflare IDs (repository variables, not secret)

Cloudflare dashboard → nex-ovia.com → **Overview** → right column "API":

| GitHub → Settings → Secrets and variables → Actions → **Variables** | Value |
|---|---|
| `CF_ACCOUNT_ID` | Account ID |
| `CF_ZONE_ID` | Zone ID |

## 2. Cloudflare API token (secret)

Cloudflare → My Profile → **API Tokens** → Create Token → *Custom token*:

- **Permissions**
  - Account · Cloudflare Pages · Edit
  - Account · Account Analytics · Read
  - Account · Account Rulesets · Read
  - Zone · Zone Settings · Edit
  - Zone · DNS · Edit
  - Zone · Zone WAF · Edit
  - Zone · Single Redirect · Edit
  - Zone · Bot Management · Edit
  - Zone · Zone · Read
  - Account · Workers R2 Storage · Edit (lets the bootstrap create the state bucket)
  - Account · Web Analytics · Edit (in some dashboards it's listed as "RUM")
- **Zone resources:** Include → Specific zone → `nex-ovia.com`
- **TTL:** 1 year (set a calendar reminder to rotate it)

Save it as repository **secret** `CLOUDFLARE_API_TOKEN`.

## 3. Terraform state bucket on R2 (free 10 GB)

1. Cloudflare → **R2** → Create bucket `nexovia-tfstate` (location: Automatic). Enabling R2 asks for a payment method even on the free tier. You stay within free usage; state is a few KB.
2. R2 → **Manage API tokens** → Create → *Object Read & Write*, **Apply to specific bucket only:** `nexovia-tfstate`.
3. Save as repository secrets `R2_ACCESS_KEY_ID` and `R2_SECRET_ACCESS_KEY`.

## 4. GitHub environments

Settings → **Environments**:

- `production`: **Required reviewers** = you. Every `terraform apply` waits for your click.
- `site`: no rules (content deploys run straight away).
- `preview`: no rules.
- `social`: holds the social secrets below (optional). Add yourself as reviewer if you want to approve every post.

## 5. First run

Push anything under `infra/` (or run the **infra** workflow manually). The plan job:

1. adopts existing Cloudflare objects with `tools/cfimport` (nothing is recreated);
2. plans with `cutover = false`, which builds Pages, security settings, rules and bots beside the live Azure site;
3. fails if any DNS record or the Pages project would be deleted (`tools/planguard`).

Approve the **apply** job in the `production` environment. The **site** workflow then deploys `dist/` to `https://nexovia-site.pages.dev` for checking.

## 6. Cutover (later, chunk C02)

Set `cutover = true` in `infra/env/prod.tfvars`, push, approve. `www` and the apex move to Cloudflare Pages. Azure stays up for 7 days; to roll back, revert that commit and approve again.
