// Command nxsite builds the nex-ovia.com site into dist/: server-rendered layout, SEO head, blog,
// feeds, security headers and share images.
//
//	nxsite build -root . -out dist -ai allow_all
//	nxsite check -out dist
package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// staticFiles are copied verbatim from the repo root.
var staticFiles = []string{"site.css", "blog.css", "shared.js", "project-graph.js", "project.toml",
	"favicon.svg", "partials/nav.html", "partials/footer.html"}

// Build renders the whole site into out. now stamps the feed when there are no posts.
func Build(root, out, aiPolicy string, now time.Time) error {
	s, err := Load(root)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	w := writer{out: out}
	s.Nav = AddLinkAfter(s.Nav, "/tools", "/blog", "Blog")
	s.Footer = AddLinkAfter(s.Footer, "/tools", "/blog", "Blog")
	c := s.Config
	base := c.Site.BaseURL

	routes := map[string]bool{"/blog/": true}
	for _, p := range s.Pages {
		routes[p.Route] = true
	}
	for _, p := range s.Posts {
		routes["/blog/"+p.Slug+"/"] = true
	}

	defaultImg := base + "/og/default.png"
	render := func(outPath, page, route string, in HeadInput) string {
		page = RenderLayout(page, s.Nav, s.Footer, route, routes)
		in.SiteName = c.Site.Name
			if in.URL == "" {
				in.URL = base + route
			}
		in.JSONLD = append([]any{SiteGraph(c)}, in.JSONLD...)
		tags, err := HeadTags(in)
		if err != nil {
			w.err = err
		}
		page = InjectHead(page, tags)
		w.file(outPath, []byte(page))
		return page
	}

	var htmlOut []string
	var sitemap []SitemapURL
	var fetchOK bool
	stats := GitHubStats(&http.Client{Timeout: 10 * time.Second}, os.Getenv("GITHUB_TOKEN"))
	for _, p := range s.Pages {
		if p.Route == "/impact/" && os.Getenv("NXSITE_OFFLINE") != "1" {
			p.HTML = FillMetrics(p.HTML, func(r string) (RepoStats, error) {
				st, err := stats(r)
				fetchOK = fetchOK || err == nil
				return st, err
			})
			if fetchOK {
				p.HTML = FillMetricDate(p.HTML, now.Format("2006-01-02"))
			}
		}
		in := HeadInput{Title: p.Title, Description: p.Description, Image: defaultImg, Type: "website"}
		// A page that declares its own canonical (e.g. a redirect page) keeps it.
		if _, rest, ok := strings.Cut(p.HTML, `<link rel="canonical" href="`); ok {
			if url, _, ok := strings.Cut(rest, `"`); ok {
				in.URL = url
			}
		}
		if p.Route == "/tools/" {
			in.JSONLD = []any{ToolsLD(c)}
		}
		htmlOut = append(htmlOut, render(p.Out, p.HTML, p.Route, in))
		if !strings.Contains(p.HTML, `name="robots" content="noindex"`) {
			sitemap = append(sitemap, SitemapURL{Loc: base + p.Route})
		}
	}

	idx, err := BlogIndexPage(c, s.Posts)
	if err != nil {
		return err
	}
	htmlOut = append(htmlOut, render("blog/index.html", idx, "/blog/", HeadInput{Title: "Blog — " + c.Site.Name,
		Description: "Writing on the Nexovia Standard, delivery sovereignty and the tools built on it.", Image: defaultImg, Type: "website"}))
	sitemap = append(sitemap, SitemapURL{Loc: base + "/blog/"})

	for _, p := range s.Posts {
		page, err := BlogPostPage(c, p)
		if err != nil {
			return fmt.Errorf("post %s: %w", p.Slug, err)
		}
		route := "/blog/" + p.Slug + "/"
		img := base + "/og/" + p.Slug + ".png"
		htmlOut = append(htmlOut, render("blog/"+p.Slug+"/index.html", page, route, HeadInput{Title: p.Title,
			Description: Teaser(p), Image: img, Type: "article", Published: p.Date,
			JSONLD: []any{PostLD(c, p, base+route, img)}}))
		sitemap = append(sitemap, SitemapURL{Loc: base + route, LastMod: p.Date.Format("2006-01-02")})
		w.card(fmt.Sprintf("og/%s.png", p.Slug), CardOG, p.Title, strings.TrimPrefix(base, "https://")+route)
		w.card(fmt.Sprintf("og/%s-ig.png", p.Slug), CardIG, p.Title, strings.TrimPrefix(base, "https://"))
	}

	// The 404 bypasses render (no SEO head), so it gets only the icon.
	notFound := InjectHead(RenderLayout(NotFoundPage(c), s.Nav, s.Footer, "/404", routes), "  "+faviconLink+"\n")
	htmlOut = append(htmlOut, notFound)
	w.file("404.html", []byte(notFound))
	w.card("og/default.png", CardOG, c.Site.Tagline, strings.TrimPrefix(base, "https://"))

	sm, err := Sitemap(sitemap)
	if err != nil {
		return err
	}
	w.file("sitemap.xml", sm)
	robots, err := Robots(base, aiPolicy)
	if err != nil {
		return err
	}
	w.file("robots.txt", []byte(robots))
	feed, err := Atom(c, s.Posts, now)
	if err != nil {
		return err
	}
	w.file("feed.xml", feed)
	w.file("llms.txt", []byte(LLMSTxt(c, s.Pages, s.Posts)))
	w.file("_headers", []byte(HeadersFile(InlineScriptHashes(htmlOut))))
	w.file("_redirects", []byte(RedirectsFile()))
	for _, f := range staticFiles {
		w.copy(filepath.Join(root, f), f)
	}
	w.copyBrand(root)
	return w.err
}

// copyBrand copies brand/ to /brand/ in the output, skipping the generator script.
// A missing brand/ directory is not an error.
func (w *writer) copyBrand(root string) {
	if w.err != nil {
		return
	}
	dir := filepath.Join(root, "brand")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	w.err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "generate-mark.py" {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		w.copy(p, filepath.ToSlash(rel))
		return w.err
	})
}

// writer collects the first error so Build reads top to bottom.
type writer struct {
	out string
	err error
}

func (w *writer) file(rel string, b []byte) {
	if w.err != nil {
		return
	}
	p := filepath.Join(w.out, filepath.FromSlash(rel))
	if w.err = os.MkdirAll(filepath.Dir(p), 0o755); w.err == nil {
		w.err = os.WriteFile(p, b, 0o644)
	}
}

func (w *writer) copy(src, rel string) {
	if w.err != nil {
		return
	}
	f, err := os.Open(src)
	if err != nil {
		w.err = err
		return
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		w.err = err
		return
	}
	w.file(rel, b)
}

func (w *writer) card(rel string, size image.Point, title, sub string) {
	if w.err != nil {
		return
	}
	b, err := Card(size, title, sub, "nex·ovia")
	if err != nil {
		w.err = err
		return
	}
	w.file(rel, b)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: nxsite build|check [flags]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	root := fs.String("root", ".", "repo root")
	out := fs.String("out", "dist", "output directory")
	ai := fs.String("ai", "allow_all", "AI crawler policy: allow_all | allow_search")
	_ = fs.Parse(os.Args[2:])
	switch os.Args[1] {
	case "build":
		if err := Build(*root, *out, *ai, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, "nxsite build:", err)
			os.Exit(1)
		}
		fmt.Println("nxsite: built", *out)
	case "check":
		problems, err := Check(*out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nxsite check:", err)
			os.Exit(1)
		}
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "  -", p)
		}
		if len(problems) > 0 {
			fmt.Fprintf(os.Stderr, "nxsite check: %d problem(s)\n", len(problems))
			os.Exit(1)
		}
		fmt.Println("nxsite check: 0 problems")
	default:
		fmt.Fprintln(os.Stderr, "unknown command", os.Args[1])
		os.Exit(2)
	}
}
