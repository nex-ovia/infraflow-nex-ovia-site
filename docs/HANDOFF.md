# Handoff: nex-ovia.com → Cloudflare (as of 2026-11-24)

Read this first when resuming. Context and rules live in `project_manifest.toml`; the full chunk plan is in `docs/plans/nexovia-site.chunks.toml`.

## Goal

Make nex-ovia.com an SEO- and AI-ready brand hub that publishes tools and blog posts, cross-posts to LinkedIn, X and Instagram, and runs on free tiers. It is hosted on Cloudflare Pages and managed by Terraform through a gated GitHub Actions pipeline.

## Where things stand

| Area | State |
|---|---|
| Terraform stack (`infra/`) | ✅ Built, 8 offline tests, tflint + checkov clean |
| Site builder (`tools/nxsite`) | ✅ Real site builds, 0 broken links, mobile + desktop clean, strict CSP |
| Social (`tools/nxsocial`, `social.yml`) | ✅ Drafts + opt-in publishing; no credentials yet (optional) |
| CI (`infra`, `site`, `security`) | ✅ Checks green on GitHub runners |
| `CF_ACCOUNT_ID` variable, `CLOUDFLARE_API_TOKEN` secret | ✅ Set by you; token verified and zone found in CI |
| GitHub environments | ✅ `production` (you approve), `site`, `preview`, `social` |
| **Terraform state in Azure** | ⏳ **Next: you run `scripts/azure-state.sh`** |
| First plan / apply | ⏳ Blocked on the step above |
| Pages preview, DNS cutover, Azure SWA removal | ⏳ C01–C04 |

The live site is still Azure Static Web Apps on `dev`, unchanged except for small mobile overflow fixes and the sponsor page now sharing the nav/footer.

## Next steps, in order

1. **Azure state (you):**
   ```bash
   git pull --ff-only origin dev
   LOCATION=eastus bash scripts/azure-state.sh
   ```
   This creates the storage account, private container and the OIDC-federated Entra app (role on the container only), and sets five `AZURE_*` repo variables. No secret is stored.
2. **Re-run infra:**
   ```bash
   gh run rerun "$(gh run list --repo nex-ovia/infraflow-nex-ovia-site --workflow infra.yml --limit 1 --json databaseId -q '.[0].databaseId')" --repo nex-ovia/infraflow-nex-ovia-site --failed
   ```
3. **Review the plan** (`cutover = false`). Expect only creates: the Pages project, zone settings, HSTS, DNSSEC, CAA, three rulesets, bot settings and Web Analytics. `planguard` must print `ok`. Existing Cloudflare objects are adopted via `tools/cfimport` import blocks, not recreated.
4. **Approve `apply`** in the `production` environment.
5. **Deploy the preview:** push any site change or re-run the **site** workflow, then verify `https://nexovia-site.pages.dev`. Check pages at 390px and 1280px, the security headers, `robots.txt`, `sitemap.xml`, `feed.xml`, `llms.txt`, and that there are no broken links.
6. **Cutover:** set `cutover = true` in `infra/env/prod.tfvars`, push, review the plan (only `www`/apex replace is allowed), and approve. Keep Azure SWA for 7 days as rollback (revert the commit and approve).
7. **Search consoles:** verify Google Search Console and Bing Webmaster via DNS TXT, then submit `https://www.nex-ovia.com/sitemap.xml`.
8. **Clean-up after 7 days:** delete the Azure SWA resource and `.github/workflows/azure-static-web-apps.yml`, and make `dev` → `main` the default if you want.
9. **Then:** add the coderev card to `/tools` (W05) and build `coderev.nex-ovia.com` (`[planned.nexovia-coderev-page]`).

## Decisions already made (don't re-litigate)

- Free tiers only. Terraform state is in **Azure Blob via GitHub OIDC**. R2 was dropped because it needs a card.
- Cloudflare needs only `CF_ACCOUNT_ID` (variable) and `CLOUDFLARE_API_TOKEN` (secret). The zone ID is looked up at runtime.
- AI crawlers are **allowed** (search, assistant and training): `ai_crawlers = "allow_all"`. Cloudflare's managed robots.txt is off and the site ships its own.
- The blog lives at `www.nex-ovia.com/blog`. Social targets are LinkedIn, X and Instagram. Drafts are always produced; publishing is per post via front matter `social.<platform> = "auto"`.
- The Blog nav link is added at build time (not in `partials/`), so the Azure site never links to a page it doesn't have.

## Gotchas learned

- This repo's deploy branch is **`dev`**. `main` is an old site.
- The Terraform registry was blocked in the cloud sandbox, so local validation there used a provider mirror. CI uses the registry normally. Provider pinned `~> 5.28`; lock file committed.
- `gitleaks dir` flags four public sample keys from X's OAuth docs in `tools/nxsocial/publish_test.go`. CI uses `gitleaks git`, where `.gitleaksignore` allowlists them by fingerprint.
- In bash with `set -o pipefail`, never `... | head -c N` inside `$(...)`; it SIGPIPEs and exits silently. `scripts/azure-state.sh` uses `od`.
- `go mod tidy` can bump to a dependency that needs a newer Go. Run with `GOTOOLCHAIN=local` and pin versions.

## Useful commands

```bash
# local build + check
cd tools && GOTOOLCHAIN=local go test ./... && go build -o ../.nxsite ./nxsite && cd .. && ./.nxsite build -out dist && ./.nxsite check -out dist

# offline infra tests
cd infra && terraform init -backend=false && terraform validate && terraform test

# CI status
gh run list --repo nex-ovia/infraflow-nex-ovia-site --limit 8
```
