package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// SitemapURL is one <url> entry.
type SitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// Sitemap renders sitemap.xml.
func Sitemap(urls []SitemapURL) ([]byte, error) {
	doc := struct {
		XMLName xml.Name     `xml:"urlset"`
		NS      string       `xml:"xmlns,attr"`
		URLs    []SitemapURL `xml:"url"`
	}{NS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls}
	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(out, '\n')...), nil
}

// AI crawlers named explicitly so the policy is readable; "*" already allows everything else.
var (
	aiSearchBots   = []string{"OAI-SearchBot", "ChatGPT-User", "Claude-SearchBot", "Claude-User", "PerplexityBot", "Perplexity-User"}
	aiTrainingBots = []string{"GPTBot", "ClaudeBot", "Google-Extended", "Applebot-Extended", "CCBot", "meta-externalagent"}
)

// Robots renders robots.txt. policy is "allow_all" or "allow_search" (training bots disallowed).
func Robots(base, policy string) (string, error) {
	if policy != "allow_all" && policy != "allow_search" {
		return "", fmt.Errorf("unknown AI crawler policy %q", policy)
	}
	var b strings.Builder
	b.WriteString("# nex-ovia.com: search engines and AI assistants are welcome to read and cite this site.\n")
	b.WriteString("User-agent: *\nAllow: /\n\n")
	b.WriteString("# AI search and assistant crawlers\n")
	for _, ua := range aiSearchBots {
		fmt.Fprintf(&b, "User-agent: %s\nAllow: /\n\n", ua)
	}
	if policy == "allow_all" {
		b.WriteString("# AI training crawlers\n")
	} else {
		b.WriteString("# AI training crawlers (not allowed)\n")
	}
	rule := map[string]string{"allow_all": "Allow: /", "allow_search": "Disallow: /"}[policy]
	for _, ua := range aiTrainingBots {
		fmt.Fprintf(&b, "User-agent: %s\n%s\n\n", ua, rule)
	}
	fmt.Fprintf(&b, "Sitemap: %s/sitemap.xml\n", base)
	return b.String(), nil
}

// Atom renders feed.xml for the given posts (newest first).
func Atom(c Config, posts []Post, now time.Time) ([]byte, error) {
	type link struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr,omitempty"`
	}
	type entry struct {
		Title   string `xml:"title"`
		Link    link   `xml:"link"`
		ID      string `xml:"id"`
		Updated string `xml:"updated"`
		Summary string `xml:"summary"`
	}
	base := c.Site.BaseURL
	updated := now
	if len(posts) > 0 {
		updated = posts[0].Date
	}
	feed := struct {
		XMLName xml.Name `xml:"feed"`
		NS      string   `xml:"xmlns,attr"`
		Title   string   `xml:"title"`
		Links   []link   `xml:"link"`
		ID      string   `xml:"id"`
		Updated string   `xml:"updated"`
		Author  struct {
			Name string `xml:"name"`
		} `xml:"author"`
		Entries []entry `xml:"entry"`
	}{NS: "http://www.w3.org/2005/Atom", Title: c.Site.Name + " blog",
		Links: []link{{Href: base + "/blog/"}, {Href: base + "/feed.xml", Rel: "self"}},
		ID:    base + "/blog/", Updated: updated.UTC().Format(time.RFC3339)}
	feed.Author.Name = c.Author.Name
	for _, p := range posts {
		u := base + "/blog/" + p.Slug + "/"
		feed.Entries = append(feed.Entries, entry{Title: p.Title, Link: link{Href: u}, ID: u,
			Updated: p.Date.UTC().Format(time.RFC3339), Summary: p.Summary})
	}
	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(out, '\n')...), nil
}

// LLMSTxt renders /llms.txt: a plain-language index of the site for AI assistants.
func LLMSTxt(c Config, pages []Page, posts []Post) string {
	base := c.Site.BaseURL
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n> %s\n\n", c.Site.Name, c.Site.Description)
	fmt.Fprintf(&b, "Author: %s. Source: %s\n\n## Pages\n\n", c.Author.Name, c.Site.Repo)
	for _, p := range pages {
		fmt.Fprintf(&b, "- [%s](%s%s): %s\n", p.Title, base, p.Route, p.Description)
	}
	if len(c.Tools) > 0 {
		b.WriteString("\n## Tools\n\n")
		for _, t := range c.Tools {
			fmt.Fprintf(&b, "- [%s](%s): %s\n", t.Name, t.URL, t.Description)
		}
	}
	if len(posts) > 0 {
		b.WriteString("\n## Blog\n\n")
		for _, p := range posts {
			fmt.Fprintf(&b, "- [%s](%s/blog/%s/): %s\n", p.Title, base, p.Slug, p.Summary)
		}
	}
	return b.String()
}
