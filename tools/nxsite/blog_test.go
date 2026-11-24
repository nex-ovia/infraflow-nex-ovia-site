package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMarkdownStripsRawHTML(t *testing.T) {
	out, err := RenderMarkdown("Hi **there**\n\n<script>alert(1)</script>\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "<script>") {
		t.Fatalf("raw HTML leaked: %s", s)
	}
	if !strings.Contains(s, "<strong>there</strong>") || !strings.Contains(s, "<table>") {
		t.Fatalf("markdown/GFM not rendered: %s", s)
	}
}

func TestBlogRenderAndIndex(t *testing.T) {
	c := testConfig()
	p := Post{Slug: "hello", Title: "Hello <World>", Date: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC),
		Tags: []string{"nxs"}, Body: "First paragraph is the teaser.\n\nSecond."}
	page, err := BlogPostPage(c, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h1 class=\"blog-title\">Hello &lt;World&gt;</h1>", `datetime="2026-11-20"`,
		`<div id="nav-placeholder"></div>`, `content="First paragraph is the teaser."`, `<span class="tag">nxs</span>`} {
		if !strings.Contains(page, want) {
			t.Errorf("post page missing %s", want)
		}
	}
	idx, err := BlogIndexPage(c, []Post{p})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(idx, `<a href="/blog/hello/">`) || strings.Count(idx, "<h1") != 1 {
		t.Errorf("index wrong:\n%s", idx)
	}
	empty, _ := BlogIndexPage(c, nil)
	if !strings.Contains(empty, "/feed.xml") {
		t.Error("empty index should point to the feed")
	}
}

func TestTeaserTruncates(t *testing.T) {
	long := strings.Repeat("word ", 80)
	if got := Teaser(Post{Body: long}); len([]rune(got)) > 200 {
		t.Fatalf("teaser too long: %d", len(got))
	}
	if Teaser(Post{Summary: "s", Body: "b"}) != "s" {
		t.Fatal("summary must win")
	}
}
