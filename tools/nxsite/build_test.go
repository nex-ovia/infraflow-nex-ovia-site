package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildFixture(t *testing.T) (root, out string) {
	t.Helper()
	return buildFixtureWith(t, nil)
}

// buildFixtureWith builds the standard fixture after mutate (if set) has changed the source tree.
func buildFixtureWith(t *testing.T, mutate func(root string)) (root, out string) {
	t.Helper()
	root = writeFixture(t)
	// The 404 page links to /about/, so the fixture needs that page for Check to pass.
	about := filepath.Join(root, "about", "index.html")
	if err := os.MkdirAll(filepath.Dir(about), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(about, []byte(`<!DOCTYPE html><html><head><title>About</title><meta name="description" content="About desc"></head>
<body><div id="nav-placeholder"></div><h1>About</h1><div id="footer-placeholder"></div></body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(root)
	}
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
	for _, f := range []string{"brand/sample.svg", "brand/generate-mark.py"} {
		p := filepath.Join(root, f)
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

func TestBuildWithoutPostsHasNoBlog(t *testing.T) {
	_, out := buildFixtureWith(t, func(root string) {
		if err := os.RemoveAll(filepath.Join(root, "content", "blog")); err != nil {
			t.Fatal(err)
		}
		nav := `<nav id="nav"><a href="/" class="nav-logo">x</a><ul class="nav-links"><li><a href="/tools">Tools</a></li></ul></nav>`
		if err := os.WriteFile(filepath.Join(root, "partials", "nav.html"), []byte(nav), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := os.Stat(filepath.Join(out, "blog", "index.html")); !os.IsNotExist(err) {
		t.Error("blog/index.html must not exist without posts")
	}
	err := filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(p) != ".html" {
			return nil
		}
		if s := read(t, p); strings.Contains(s, `href="/blog/`) || strings.Contains(s, `href="/blog"`) {
			t.Errorf("%s links to /blog without posts", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if sm := read(t, filepath.Join(out, "sitemap.xml")); strings.Contains(sm, "/blog") {
		t.Error("sitemap lists blog without posts")
	}
	if nf := read(t, filepath.Join(out, "404.html")); !strings.Contains(nf, `href="/about/"`) {
		t.Error("404 page should link to the about page")
	}
}

func TestBuildWithPostHasBlogLinks(t *testing.T) {
	_, out := buildFixture(t)
	if _, err := os.Stat(filepath.Join(out, "blog", "index.html")); err != nil {
		t.Error("blog/index.html missing with a post")
	}
	if home := read(t, filepath.Join(out, "index.html")); !strings.Contains(home, `href="/blog`) {
		t.Error("nav/footer blog link missing with a post")
	}
}

func TestBuildCopiesBrandExceptGenerator(t *testing.T) {
	_, out := buildFixture(t)
	if _, err := os.Stat(filepath.Join(out, "brand", "sample.svg")); err != nil {
		t.Error("brand/sample.svg not copied")
	}
	if _, err := os.Stat(filepath.Join(out, "brand", "generate-mark.py")); !os.IsNotExist(err) {
		t.Error("brand/generate-mark.py must not be copied")
	}
}

func TestBuildWithoutBrandDirSucceeds(t *testing.T) {
	root := writeFixture(t)
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
	if _, err := os.Stat(filepath.Join(root, "brand")); !os.IsNotExist(err) {
		t.Fatal("fixture must not contain brand/")
	}
	out := filepath.Join(t.TempDir(), "dist")
	if err := Build(root, out, "allow_all", time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("build without brand/ failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "brand")); !os.IsNotExist(err) {
		t.Error("brand/ must not be created when source is absent")
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
