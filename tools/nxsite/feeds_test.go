package main

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestSitemapValidXML(t *testing.T) {
	out, err := Sitemap([]SitemapURL{{Loc: "https://www.example.com/", LastMod: "2026-11-24"}, {Loc: "https://www.example.com/a&b/"}})
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		URLs []SitemapURL `xml:"url"`
	}
	if err := xml.Unmarshal(out, &v); err != nil || len(v.URLs) != 2 || v.URLs[1].Loc != "https://www.example.com/a&b/" {
		t.Fatalf("bad sitemap (%v): %s", err, out)
	}
}

func TestRobotsPolicies(t *testing.T) {
	all, err := Robots("https://www.example.com", "allow_all")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(all, "Disallow") || !strings.Contains(all, "Sitemap: https://www.example.com/sitemap.xml") {
		t.Fatalf("allow_all wrong:\n%s", all)
	}
	search, _ := Robots("https://www.example.com", "allow_search")
	if !strings.Contains(search, "User-agent: GPTBot\nDisallow: /") || !strings.Contains(search, "User-agent: OAI-SearchBot\nAllow: /") {
		t.Fatalf("allow_search wrong:\n%s", search)
	}
	if _, err := Robots("x", "block"); err == nil {
		t.Fatal("unknown policy should error")
	}
}

func TestAtomFeedValidXML(t *testing.T) {
	posts := []Post{{Slug: "b", Title: "B & C", Summary: "s", Date: time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)}}
	out, err := Atom(testConfig(), posts, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Updated string `xml:"updated"`
		Entries []struct {
			Title string `xml:"title"`
			ID    string `xml:"id"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(out, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Entries) != 1 || v.Entries[0].Title != "B & C" || v.Entries[0].ID != "https://www.example.com/blog/b/" || v.Updated != "2026-11-22T00:00:00Z" {
		t.Fatalf("bad feed: %+v", v)
	}
}

func TestLLMSTxtListsPagesToolsPosts(t *testing.T) {
	out := LLMSTxt(testConfig(), []Page{{Title: "Tools", Route: "/tools/", Description: "d"}},
		[]Post{{Slug: "b", Title: "B", Summary: "s"}})
	for _, want := range []string{"# Ex", "[Tools](https://www.example.com/tools/)", "[t1](https://t1.example.com)", "/blog/b/"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}
