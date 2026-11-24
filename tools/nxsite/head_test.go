package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

func testConfig() Config {
	var c Config
	c.Site.BaseURL = "https://www.example.com"
	c.Site.Name = "Ex"
	c.Site.Description = "Example"
	c.Site.Repo = "https://github.com/ex"
	c.Author.Name = "A. Person"
	c.Author.SameAs = []string{"https://github.com/a"}
	c.Tools = append(c.Tools, struct {
		Name        string `toml:"name"`
		URL         string `toml:"url"`
		Description string `toml:"description"`
	}{"t1", "https://t1.example.com", "Tool one"})
	return c
}

var ldRe = regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`)

func TestHeadTagsAndJSONLDValid(t *testing.T) {
	c := testConfig()
	tags, err := HeadTags(HeadInput{Title: `A "quoted" title`, Description: "d", URL: "https://www.example.com/tools/",
		Image: "https://www.example.com/og/default.png", Type: "website", SiteName: "Ex",
		JSONLD: []any{SiteGraph(c), ToolsLD(c)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<link rel="canonical" href="https://www.example.com/tools/">`,
		`<meta property="og:title" content="A &#34;quoted&#34; title">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta property="og:image:width" content="1200">`,
		`<link rel="icon" href="/favicon.svg" type="image/svg+xml">`,
	} {
		if !strings.Contains(tags, want) {
			t.Errorf("missing %s", want)
		}
	}
	blocks := ldRe.FindAllStringSubmatch(tags, -1)
	if len(blocks) != 2 {
		t.Fatalf("want 2 JSON-LD blocks, got %d", len(blocks))
	}
	for _, b := range blocks {
		var v map[string]any
		if err := json.Unmarshal([]byte(b[1]), &v); err != nil {
			t.Fatalf("invalid JSON-LD: %v", err)
		}
		if v["@context"] != "https://schema.org" {
			t.Errorf("missing @context: %v", v)
		}
	}
}

func TestInjectHeadReplacesStaleTags(t *testing.T) {
	page := `<html><head><title>x</title>
  <meta property="og:url" content="https://example.com">
  <link rel="canonical" href="https://old">
</head><body></body></html>`
	out := InjectHead(page, `  <link rel="canonical" href="https://new">`+"\n")
	if strings.Contains(out, "https://old") || strings.Contains(out, `content="https://example.com"`) {
		t.Fatalf("stale tags kept:\n%s", out)
	}
	if strings.Count(out, `rel="canonical"`) != 1 || !strings.Contains(out, "https://new") {
		t.Fatalf("new canonical missing:\n%s", out)
	}
}

func TestPostLDFields(t *testing.T) {
	p := Post{Title: "T", Summary: "S", Date: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), Tags: []string{"a", "b"}}
	ld := PostLD(testConfig(), p, "https://www.example.com/blog/t/", "https://www.example.com/og/t.png")
	if ld["@type"] != "BlogPosting" || ld["datePublished"] != "2026-11-20" || ld["keywords"] != "a, b" {
		t.Fatalf("bad BlogPosting: %v", ld)
	}
}

func TestHeadEscapesScriptCloseInJSONLD(t *testing.T) {
	tags, err := HeadTags(HeadInput{URL: "u", Type: "website", JSONLD: []any{map[string]string{"x": "</script><b>"}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(tags, "</script>") != 1 {
		t.Fatalf("script close not escaped: %s", tags)
	}
}
