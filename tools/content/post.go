// Package content holds the blog post model shared by nxsite (rendering) and nxsocial (syndication).
package content

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Post is one Markdown blog post.
type Post struct {
	Slug    string
	Title   string    `toml:"title"`
	Date    time.Time `toml:"date"`
	Summary string    `toml:"summary"`
	Tags    []string  `toml:"tags"`
	Draft   bool      `toml:"draft"`
	Social  struct {
		LinkedIn  string `toml:"linkedin"`
		X         string `toml:"x"`
		Instagram string `toml:"instagram"`
	} `toml:"social"`
	Body string // markdown
}

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ParsePost splits +++ TOML front matter from the Markdown body and validates required fields.
func ParsePost(slug string, raw []byte) (Post, error) {
	var p Post
	if !slugRe.MatchString(slug) {
		return p, fmt.Errorf("slug %q must be lowercase words joined by hyphens", slug)
	}
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(raw, []byte("+++\n")) {
		return p, fmt.Errorf("missing +++ front matter")
	}
	rest := raw[4:]
	end := bytes.Index(rest, []byte("\n+++"))
	if end < 0 {
		return p, fmt.Errorf("unterminated front matter")
	}
	if _, err := toml.Decode(string(rest[:end]), &p); err != nil {
		return p, fmt.Errorf("front matter: %w", err)
	}
	p.Slug = slug
	p.Body = strings.TrimLeft(string(rest[end+4:]), "\n")
	switch {
	case p.Title == "":
		return p, fmt.Errorf("title is required")
	case p.Date.IsZero():
		return p, fmt.Errorf("date is required")
	}
	for _, v := range []*string{&p.Social.LinkedIn, &p.Social.X, &p.Social.Instagram} {
		if *v == "" {
			*v = "draft"
		}
		if *v != "off" && *v != "draft" && *v != "auto" {
			return p, fmt.Errorf("social values must be off, draft or auto (got %q)", *v)
		}
	}
	return p, nil
}

// Teaser returns the summary, or the first paragraph of the body when summary is empty.
func Teaser(p Post) string {
	if p.Summary != "" {
		return p.Summary
	}
	para := strings.SplitN(strings.TrimSpace(p.Body), "\n\n", 2)[0]
	para = strings.NewReplacer("**", "", "__", "", "`", "", "#", "").Replace(para)
	if len(para) > 200 {
		para = strings.TrimSpace(para[:197]) + "…"
	}
	return strings.TrimSpace(para)
}
