package main

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// HeadInput describes the SEO metadata for one URL.
type HeadInput struct {
	Title       string
	Description string
	URL         string // absolute canonical URL
	Image       string // absolute share-image URL (1200x630)
	Type        string // "website" or "article"
	Published   time.Time
	SiteName    string
	JSONLD      []any
}

// staleHeadRe matches tags HeadTags owns, so existing hand-written copies are replaced, not duplicated.
var staleHeadRe = regexp.MustCompile(`(?i)\s*<(?:meta\s+(?:property|name)="(?:og:[^"]*|twitter:[^"]*)"[^>]*|link\s+rel="canonical"[^>]*|link\s+rel="alternate"\s+type="application/atom\+xml"[^>]*)>`)

// faviconLink is the SVG icon every page carries (favicon.svg is copied to the output root).
const faviconLink = `<link rel="icon" href="/favicon.svg" type="image/svg+xml">`

// HeadTags renders the icon, canonical, Open Graph, Twitter card, feed link and JSON-LD tags.
func HeadTags(in HeadInput) (string, error) {
	esc := html.EscapeString
	var b strings.Builder
	line := func(format string, a ...any) { fmt.Fprintf(&b, "  "+format+"\n", a...) }
	line(faviconLink)
	line(`<link rel="canonical" href="%s">`, esc(in.URL))
	line(`<link rel="alternate" type="application/atom+xml" title="%s" href="/feed.xml">`, esc(in.SiteName))
	line(`<meta property="og:site_name" content="%s">`, esc(in.SiteName))
	line(`<meta property="og:type" content="%s">`, esc(in.Type))
	line(`<meta property="og:title" content="%s">`, esc(in.Title))
	line(`<meta property="og:description" content="%s">`, esc(in.Description))
	line(`<meta property="og:url" content="%s">`, esc(in.URL))
	if in.Image != "" {
		line(`<meta property="og:image" content="%s">`, esc(in.Image))
		line(`<meta property="og:image:width" content="1200">`)
		line(`<meta property="og:image:height" content="630">`)
	}
	if in.Type == "article" && !in.Published.IsZero() {
		line(`<meta property="article:published_time" content="%s">`, in.Published.Format(time.RFC3339))
	}
	line(`<meta name="twitter:card" content="summary_large_image">`)
	line(`<meta name="twitter:title" content="%s">`, esc(in.Title))
	line(`<meta name="twitter:description" content="%s">`, esc(in.Description))
	if in.Image != "" {
		line(`<meta name="twitter:image" content="%s">`, esc(in.Image))
	}
	for _, ld := range in.JSONLD {
		j, err := json.Marshal(ld)
		if err != nil {
			return "", err
		}
		// "</" can't appear in JSON-LD inside <script>; escape it.
		line(`<script type="application/ld+json">%s</script>`, strings.ReplaceAll(string(j), "</", `<\/`))
	}
	return b.String(), nil
}

// InjectHead removes stale SEO tags and inserts tags just before </head>.
func InjectHead(page, tags string) string {
	page = staleHeadRe.ReplaceAllString(page, "")
	i := strings.Index(strings.ToLower(page), "</head>")
	if i < 0 {
		return page
	}
	return page[:i] + tags + page[i:]
}

// SiteGraph is the Organization + WebSite + Person JSON-LD shared by every page.
func SiteGraph(c Config) map[string]any {
	base := c.Site.BaseURL
	person := map[string]any{"@type": "Person", "@id": base + "/#author", "name": c.Author.Name, "url": orDefault(c.Author.URL, base)}
	if len(c.Author.SameAs) > 0 {
		person["sameAs"] = c.Author.SameAs
	}
	org := map[string]any{"@type": "Organization", "@id": base + "/#org", "name": c.Site.Name, "url": base + "/",
		"founder": map[string]any{"@id": base + "/#author"}}
	if c.Site.Repo != "" {
		org["sameAs"] = []string{c.Site.Repo}
	}
	site := map[string]any{"@type": "WebSite", "@id": base + "/#website", "name": c.Site.Name, "url": base + "/",
		"description": c.Site.Description, "inLanguage": orDefault(c.Site.Language, "en"), "publisher": map[string]any{"@id": base + "/#org"}}
	return map[string]any{"@context": "https://schema.org", "@graph": []any{org, site, person}}
}

// ToolsLD lists each tool as a SoftwareApplication.
func ToolsLD(c Config) map[string]any {
	var apps []any
	for _, t := range c.Tools {
		apps = append(apps, map[string]any{"@type": "SoftwareApplication", "name": t.Name, "url": t.URL,
			"description": t.Description, "applicationCategory": "DeveloperApplication",
			"operatingSystem": "macOS, Linux, Windows", "offers": map[string]any{"@type": "Offer", "price": "0", "priceCurrency": "USD"},
			"publisher": map[string]any{"@id": c.Site.BaseURL + "/#org"}})
	}
	return map[string]any{"@context": "https://schema.org", "@graph": apps}
}

// PostLD is the BlogPosting for one post.
func PostLD(c Config, p Post, url, image string) map[string]any {
	ld := map[string]any{"@context": "https://schema.org", "@type": "BlogPosting", "headline": p.Title,
		"description": p.Summary, "datePublished": p.Date.Format("2006-01-02"), "url": url, "mainEntityOfPage": url,
		"author": map[string]any{"@id": c.Site.BaseURL + "/#author"}, "publisher": map[string]any{"@id": c.Site.BaseURL + "/#org"}}
	if image != "" {
		ld["image"] = image
	}
	if len(p.Tags) > 0 {
		ld["keywords"] = strings.Join(p.Tags, ", ")
	}
	return ld
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
