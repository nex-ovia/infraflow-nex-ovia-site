package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildFixture(t *testing.T) (root, out string) {
	t.Helper()
	root = writeFixture(t)
	for _, f := range staticFiles {
		p := filepath.Join(root, f)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("/* "+f+" */"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out = filepath.Join(t.TempDir(), "dist")
	if err := Build(root, out, "allow_all", time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	return root, out
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildProducesSEOReadySite(t *testing.T) {
	_, out := buildFixture(t)
	for _, f := range []string{"index.html", "tools/index.html", "blog/index.html",
		"blog/first-post/index.html", "404.html", "sitemap.xml", "robots.txt", "feed.xml", "llms.txt",
		"_headers", "_redirects", "og/default.png", "og/first-post.png", "og/first-post-ig.png", "site.css", "favicon.svg"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	home := read(t, filepath.Join(out, "index.html"))
	if !strings.Contains(home, `<nav id="nav">`) || strings.Contains(home, "nav-placeholder") {
		t.Error("nav not server-rendered on home")
	}
	if !strings.Contains(home, `<link rel="canonical" href="https://www.example.com/">`) || strings.Contains(home, `content="https://example.com"`) {
		t.Error("home canonical/og:url wrong")
	}
	icon := `<link rel="icon" href="/favicon.svg" type="image/svg+xml">`
	for _, f := range []string{"index.html", "tools/index.html", "blog/index.html", "blog/first-post/index.html", "404.html"} {
		if !strings.Contains(read(t, filepath.Join(out, f)), icon) {
			t.Errorf("%s missing favicon link", f)
		}
	}
	tools := read(t, filepath.Join(out, "tools/index.html"))
	if !strings.Contains(tools, `"SoftwareApplication"`) || !strings.Contains(tools, `<li class="active"><a href="/tools/">`) {
		t.Error("tools page missing JSON-LD or active nav")
	}
	if h := read(t, filepath.Join(out, "_headers")); !strings.Contains(h, "'sha256-") {
		t.Error("inline script hash missing from CSP")
	}
	if sm := read(t, filepath.Join(out, "sitemap.xml")); !strings.Contains(sm, "https://www.example.com/blog/first-post/") || strings.Contains(sm, "hidden") {
		t.Error("sitemap should list published post only")
	}
}

func TestCheckPassesBuiltSiteAndCatchesBreaks(t *testing.T) {
	_, out := buildFixture(t)
	problems, err := Check(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("built fixture should be clean, got %v", problems)
	}
	bad := `<html><head><meta name="description" content="d"></head><body><h1>a</h1><h1>b</h1><a href="/nope/">x</a></body></html>`
	if err := os.WriteFile(filepath.Join(out, "bad.html"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, _ = Check(out)
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"bad.html: 2 <h1>", "bad.html: missing canonical", "bad.html: broken link /nope/"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Check missed %q in %v", want, problems)
		}
	}
}
