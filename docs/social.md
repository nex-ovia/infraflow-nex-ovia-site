# Social syndication (LinkedIn, X, Instagram)

Publish on nex-ovia.com first; social posts link back to it.

## Writing a post

Copy `content/blog/_example.md` to `content/blog/<slug>.md` (lowercase words joined by hyphens), set `draft = false`, and choose per platform:

```toml
[social]
linkedin  = "auto"   # publish via API when the post goes live
x         = "draft"  # only produce copy in the workflow artifact
instagram = "off"    # nothing
```

When you push to `dev`, the **social** workflow:

- builds drafts for all three platforms, uploaded as the `social-drafts` artifact on the run;
- waits until the post URL is live;
- publishes platforms set to `auto` that have credentials.

Only posts **added** in that push are processed, so editing an old post never re-posts it. To redo one post, run the workflow manually with its slug.

## Credentials (GitHub → Settings → Environments → `social` → secrets)

All three are free developer products. Each step is done once.

### LinkedIn (personal profile)

1. <https://www.linkedin.com/developers/apps> → Create app (needs a LinkedIn Page; your own page is fine).
2. Products → add **Share on LinkedIn** and **Sign In with LinkedIn using OpenID Connect**.
3. Use the OAuth token generator (Auth tab) with scopes `openid profile w_member_social` to get an access token.
4. Call `GET https://api.linkedin.com/v2/userinfo` with it. The `sub` value is your member id.
5. Secrets: `LINKEDIN_TOKEN` and `LINKEDIN_AUTHOR_URN` = `urn:li:person:<sub>`.
6. Tokens last **60 days**. Set repository variable `LINKEDIN_TOKEN_EXPIRES` (YYYY-MM-DD); the workflow opens an issue 10 days before expiry.

### X

1. <https://developer.x.com> → Free plan → Project + App.
2. User authentication settings → **Read and write**.
3. Keys and tokens → API Key/Secret, then generate the **Access Token and Secret** for your account. Do this *after* switching to read-write.
4. Secrets: `X_API_KEY`, `X_API_SECRET`, `X_ACCESS_TOKEN`, `X_ACCESS_SECRET`. These don't expire.
5. The free tier allows only a small number of posts per month. A long teaser becomes a short thread, and each part counts.

### Instagram

1. Switch the Instagram account to **Professional (Creator or Business)** and link it to a Facebook Page.
2. <https://developers.facebook.com> → Create app (type Business) → add **Instagram Graph API**.
3. In Graph API Explorer, get a user token with `instagram_basic`, `instagram_content_publish` and `pages_show_list`, then exchange it for a **long-lived token**. For a non-expiring token, use a *System User* in Meta Business Settings.
4. Find your Instagram user id: `GET /me/accounts` → page → `?fields=instagram_business_account`.
5. Secrets: `IG_USER_ID` and `IG_ACCESS_TOKEN`. Optionally set `IG_GRAPH_VERSION` (default `v23.0`).
6. Each post uses the generated 1080×1350 card (`/og/<slug>-ig.png`). Instagram can't link out, so the caption says "link in bio". Put nex-ovia.com/blog in your bio.

## Notes

- API versions (`LINKEDIN_VERSION` default `202510`, `IG_GRAPH_VERSION` default `v23.0`) move over time. Set the environment variable to a newer version when a platform deprecates the default.
- A failed platform doesn't block the others; the run is marked failed so you notice.
