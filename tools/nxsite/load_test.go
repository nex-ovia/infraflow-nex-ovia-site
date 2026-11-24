package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture builds a minimal site tree and returns its root.
func writeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"content/site.toml": `[site]
base_url = "https://www.example.com/"
name = "Ex"
description = "Example site"
[author]
name = "A. Person"
same_as = ["https://github.com/a"]
[[tool]]
name = "t1"
url = "https://t1.example.com"
description = "Tool one"
`,
		"partials/nav.html":    `<nav id="nav"><a href="/" class="nav-logo">x</a><ul class="nav-links"><li><a href="/tools">Tools</a></li><li><a href="/sponser">S</a></li></ul></nav>`,
		"partials/footer.html": `<footer><a href="/tools">Tools</a></footer>`,
		"index.html": `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Home &amp; more</title>
<meta name="description" content="Home desc"><meta property="og:url" content="https://example.com"></head>
<body><div id="nav-placeholder"></div><h1>Home</h1><a href="/tools">t</a><script>console.log(1)</script></body></html>`,
		"tools/index.html": `<!DOCTYPE html><html><head><title>Tools</title><meta name="description" content="Tools desc"></head>
<body><div id="nav-placeholder"></div><h1>Tools</h1><div id="footer-placeholder"></div></body></html>`,
		"sponser/index.html": `<!DOCTYPE html><html><head><title>Sponsor</title><meta name="description" content="Sponsor desc"></head>
<body><div id="nav-placeholder"></div><h1>Sponsor</h1></body></html>`,
		"content/blog/first-post.md": "+++\ntitle = \"First\"\ndate = 2026-11-20\nsummary = \"Hello\"\ntags = [\"a\"]\n[social]\nx = \"auto\"\n+++\n\nBody **bold**.\n",
		"content/blog/hidden.md":     "+++\ntitle = \"Hidden\"\ndate = 2026-11-21\ndraft = true\n+++\nx\n",
		"content/blog/_example.md":   "not parsed",
	}
	for p, c := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLoadPagesRoutesAndRename(t *testing.T) {
	s, err := Load(writeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.Config.Site.BaseURL != "https://www.example.com" {
		t.Errorf("base_url not trimmed: %q", s.Config.Site.BaseURL)
	}
	got := map[string]Page{}
	for _, p := range s.Pages {
		got[p.Route] = p
	}
	if got["/"].Title != "Home & more" || got["/"].Description != "Home desc" {
		t.Errorf("home meta wrong: %+v", got["/"])
	}
	if got["/sponsor/"].Out != "sponsor/index.html" || got["/sponsor/"].Src != "sponser/index.html" {
		t.Errorf("sponser not renamed: %+v", got["/sponsor/"])
	}
	if len(s.Pages) != 3 {
		t.Errorf("want 3 pages, got %d", len(s.Pages))
	}
}

func TestLoadSkipsDraftsAndUnderscoreFiles(t *testing.T) {
	s, err := Load(writeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Posts) != 1 || s.Posts[0].Slug != "first-post" {
		t.Fatalf("want only first-post, got %+v", s.Posts)
	}
	p := s.Posts[0]
	if p.Social.X != "auto" || p.Social.LinkedIn != "draft" || !strings.HasPrefix(p.Body, "Body") {
		t.Errorf("post parsed wrong: %+v", p)
	}
}

func TestParsePostValidation(t *testing.T) {
	cases := map[string]struct{ slug, raw string }{
		"bad slug":       {"Bad_Slug", "+++\ntitle=\"t\"\ndate=2026-01-01\n+++\n"},
		"no front":       {"ok", "just text"},
		"unterminated":   {"ok", "+++\ntitle=\"t\"\n"},
		"no title":       {"ok", "+++\ndate=2026-01-01\n+++\n"},
		"no date":        {"ok", "+++\ntitle=\"t\"\n+++\n"},
		"bad social val": {"ok", "+++\ntitle=\"t\"\ndate=2026-01-01\n[social]\nx=\"yes\"\n+++\n"},
	}
	for name, c := range cases {
		if _, err := ParsePost(c.slug, []byte(c.raw)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
