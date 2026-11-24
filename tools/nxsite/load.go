package main

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Site is everything the build needs, loaded from the repo root.
type Site struct {
	Config Config
	Pages  []Page
	Posts  []Post
	Nav    string // partials/nav.html
	Footer string // partials/footer.html
}

type Config struct {
	Site struct {
		BaseURL     string `toml:"base_url"`
		Name        string `toml:"name"`
		Tagline     string `toml:"tagline"`
		Description string `toml:"description"`
		Language    string `toml:"language"`
		Repo        string `toml:"repo"`
	} `toml:"site"`
	Author struct {
		Name   string   `toml:"name"`
		URL    string   `toml:"url"`
		SameAs []string `toml:"same_as"`
	} `toml:"author"`
	Tools []struct {
		Name        string `toml:"name"`
		URL         string `toml:"url"`
		Description string `toml:"description"`
	} `toml:"tool"`
}

// Page is one existing hand-written HTML page.
type Page struct {
	Src         string // path relative to root, e.g. "tools/index.html"
	Out         string // output path relative to dist
	Route       string // canonical path with trailing slash, e.g. "/tools/"
	Title       string
	Description string
	HTML        string
}

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

// renamedDirs maps a source directory to its published name (fixes the old /sponser typo).
var renamedDirs = map[string]string{"sponser": "sponsor"}

var (
	titleRe = regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	descRe  = regexp.MustCompile(`(?is)<meta\s+name="description"\s+content="([^"]*)"`)
)

// Load reads site.toml, partials, every */index.html page and content/blog/*.md.
func Load(root string) (*Site, error) {
	s := &Site{}
	if _, err := toml.DecodeFile(filepath.Join(root, "content", "site.toml"), &s.Config); err != nil {
		return nil, fmt.Errorf("content/site.toml: %w", err)
	}
	if s.Config.Site.BaseURL == "" {
		return nil, fmt.Errorf("content/site.toml: site.base_url is required")
	}
	s.Config.Site.BaseURL = strings.TrimRight(s.Config.Site.BaseURL, "/")
	var err error
	if s.Nav, err = readString(filepath.Join(root, "partials", "nav.html")); err != nil {
		return nil, err
	}
	if s.Footer, err = readString(filepath.Join(root, "partials", "footer.html")); err != nil {
		return nil, err
	}
	if s.Pages, err = loadPages(root); err != nil {
		return nil, err
	}
	if s.Posts, err = loadPosts(filepath.Join(root, "content", "blog")); err != nil {
		return nil, err
	}
	return s, nil
}

func readString(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func loadPages(root string) ([]Page, error) {
	files, err := filepath.Glob(filepath.Join(root, "*", "index.html"))
	if err != nil {
		return nil, err
	}
	files = append([]string{filepath.Join(root, "index.html")}, files...)
	var pages []Page
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "partials/") || strings.HasPrefix(rel, "dist/") {
			continue
		}
		p, err := loadPage(f, rel)
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Route < pages[j].Route })
	return pages, nil
}

func loadPage(path, rel string) (Page, error) {
	src, err := readString(path)
	if err != nil {
		return Page{}, err
	}
	dir := strings.TrimSuffix(rel, "index.html")
	dir = strings.TrimSuffix(dir, "/")
	if n, ok := renamedDirs[dir]; ok {
		dir = n
	}
	p := Page{Src: rel, HTML: src, Route: "/", Out: "index.html"}
	if dir != "" {
		p.Route = "/" + dir + "/"
		p.Out = dir + "/index.html"
	}
	m := titleRe.FindStringSubmatch(src)
	if m == nil {
		return Page{}, fmt.Errorf("%s: missing <title>", rel)
	}
	p.Title = html.UnescapeString(strings.TrimSpace(m[1]))
	if d := descRe.FindStringSubmatch(src); d != nil {
		p.Description = html.UnescapeString(d[1])
	}
	return p, nil
}

func loadPosts(dir string) ([]Post, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	var posts []Post
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasPrefix(base, "_") {
			continue // templates/examples
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		p, err := ParsePost(strings.TrimSuffix(base, ".md"), raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base, err)
		}
		if !p.Draft {
			posts = append(posts, p)
		}
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].Date.After(posts[j].Date) })
	return posts, nil
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
