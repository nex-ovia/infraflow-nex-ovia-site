// Command nxsocial turns new blog posts into LinkedIn / X / Instagram drafts and, when a post opts in
// (front matter social.<platform> = "auto") and credentials are present, publishes them.
//
//	nxsocial -root . -base https://www.nex-ovia.com -slugs my-post -out drafts           # drafts only
//	nxsocial -root . -base https://www.nex-ovia.com -slugs my-post -out drafts -publish  # + opt-in posting
//
// Credentials (environment, from the GitHub "social" environment secrets):
//
//	LINKEDIN_TOKEN, LINKEDIN_AUTHOR_URN [, LINKEDIN_VERSION]
//	X_API_KEY, X_API_SECRET, X_ACCESS_TOKEN, X_ACCESS_SECRET
//	IG_USER_ID, IG_ACCESS_TOKEN [, IG_GRAPH_VERSION]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nex-ovia/infraflow-nex-ovia-site/tools/content"
)

// Mode returns the post's setting for a platform: off, draft or auto.
func Mode(p content.Post, platform string) string {
	switch platform {
	case "linkedin":
		return p.Social.LinkedIn
	case "x":
		return p.Social.X
	case "instagram":
		return p.Social.Instagram
	}
	return "off"
}

// Publishers builds the publishers whose credentials are present in env.
func Publishers(env func(string) string, c *http.Client) map[string]Publisher {
	out := map[string]Publisher{}
	if env("LINKEDIN_TOKEN") != "" && env("LINKEDIN_AUTHOR_URN") != "" {
		out["linkedin"] = LinkedIn{Base: "https://api.linkedin.com", Token: env("LINKEDIN_TOKEN"),
			Author: env("LINKEDIN_AUTHOR_URN"), Version: orEnv(env, "LINKEDIN_VERSION", "202510"), HTTP: c}
	}
	if env("X_API_KEY") != "" && env("X_API_SECRET") != "" && env("X_ACCESS_TOKEN") != "" && env("X_ACCESS_SECRET") != "" {
		out["x"] = X{Base: "https://api.x.com", ConsumerKey: env("X_API_KEY"), ConsumerSecret: env("X_API_SECRET"),
			Token: env("X_ACCESS_TOKEN"), TokenSec: env("X_ACCESS_SECRET"), HTTP: c}
	}
	if env("IG_USER_ID") != "" && env("IG_ACCESS_TOKEN") != "" {
		out["instagram"] = Instagram{Base: "https://graph.facebook.com", Version: orEnv(env, "IG_GRAPH_VERSION", "v23.0"),
			UserID: env("IG_USER_ID"), Token: env("IG_ACCESS_TOKEN"), HTTP: c}
	}
	return out
}

func orEnv(env func(string) string, k, d string) string {
	if v := env(k); v != "" {
		return v
	}
	return d
}

// Result is the outcome for one platform.
type Result struct {
	Platform, Status, ID string // Status: published | draft | skipped | failed
	Err                  error
}

// Run publishes one post's drafts to every platform set to "auto" with credentials available.
// Platforms set to "draft" (or without credentials) are reported as draft; "off" as skipped.
func Run(ctx context.Context, p content.Post, d Drafts, pubs map[string]Publisher, publish bool) []Result {
	var out []Result
	for _, pl := range []string{"linkedin", "x", "instagram"} {
		mode := Mode(p, pl)
		pub, ok := pubs[pl]
		switch {
		case mode == "off":
			out = append(out, Result{Platform: pl, Status: "skipped"})
		case mode != "auto" || !publish:
			out = append(out, Result{Platform: pl, Status: "draft"})
		case !ok:
			out = append(out, Result{Platform: pl, Status: "draft", Err: errors.New("auto requested but no credentials configured")})
		default:
			id, err := pub.Publish(ctx, d)
			if err != nil {
				out = append(out, Result{Platform: pl, Status: "failed", Err: err})
			} else {
				out = append(out, Result{Platform: pl, Status: "published", ID: id})
			}
		}
	}
	return out
}

// WaitLive polls url until it returns 200 or the deadline passes (the site deploy may still be running).
func WaitLive(ctx context.Context, c *http.Client, url string, tries int, every time.Duration) error {
	var last string
	for i := 0; i < tries; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		resp, err := c.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			last = resp.Status
		} else {
			last = err.Error()
		}
		if i+1 < tries {
			time.Sleep(every)
		}
	}
	return fmt.Errorf("%s not live after %d checks (last: %s)", url, tries, last)
}

func main() {
	root := flag.String("root", ".", "repo root")
	base := flag.String("base", "https://www.nex-ovia.com", "public site URL")
	slugs := flag.String("slugs", "", "comma-separated post slugs")
	out := flag.String("out", "drafts", "directory for draft files")
	publish := flag.Bool("publish", false, "publish platforms set to auto (otherwise drafts only)")
	flag.Parse()

	ctx := context.Background()
	client := &http.Client{Timeout: 30 * time.Second}
	pubs := Publishers(os.Getenv, client)
	failed := false
	for _, slug := range strings.Split(*slugs, ",") {
		if slug = strings.TrimSpace(slug); slug == "" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(*root, "content", "blog", slug+".md"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "nxsocial:", err)
			failed = true
			continue
		}
		p, err := content.ParsePost(slug, raw)
		if err != nil || p.Draft {
			fmt.Fprintf(os.Stderr, "nxsocial: %s: skipped (%v, draft=%v)\n", slug, err, p.Draft)
			continue
		}
		d := MakeDrafts(p, *base)
		if err := os.MkdirAll(*out, 0o755); err == nil {
			err = os.WriteFile(filepath.Join(*out, slug+".txt"), []byte(d.Render()), 0o644)
		}
		if *publish {
			if err := WaitLive(ctx, client, d.URL, 20, 30*time.Second); err != nil {
				fmt.Fprintln(os.Stderr, "nxsocial:", err, "— not publishing", slug)
				failed = true
				continue
			}
		}
		for _, r := range Run(ctx, p, d, pubs, *publish) {
			line := fmt.Sprintf("%s %-9s %-9s %s", slug, r.Platform, r.Status, r.ID)
			if r.Err != nil {
				line += " (" + r.Err.Error() + ")"
			}
			fmt.Println(line)
			if r.Status == "failed" {
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}
