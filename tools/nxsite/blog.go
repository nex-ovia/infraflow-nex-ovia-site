package main

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/nex-ovia/infraflow-nex-ovia-site/tools/content"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// md renders GitHub-flavoured Markdown. Raw HTML in posts is NOT rendered (goldmark default), so a
// post can't inject scripts and the CSP stays strict.
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Typographer),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

// RenderMarkdown converts a post body to safe HTML.
func RenderMarkdown(src string) (template.HTML, error) {
	var b bytes.Buffer
	if err := md.Convert([]byte(src), &b); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil // goldmark escapes raw HTML by default
}

// Teaser delegates to the shared content package.
func Teaser(p Post) string { return content.Teaser(p) }

const shellTop = `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <meta name="description" content="{{.Description}}">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=DM+Serif+Display:ital@0;1&family=DM+Sans:opsz,wght@9..40,300;9..40,400;9..40,500;9..40,600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/site.css">
  <link rel="stylesheet" href="/blog.css">
</head>
<body>
  <div id="nav-placeholder"></div>
`

const shellBottom = `
  <div id="footer-placeholder"></div>
  <script src="/shared.js"></script>
</body>
</html>
`

var postTmpl = template.Must(template.New("post").Parse(shellTop + `
  <main class="blog-main">
    <article class="container blog-article">
      <p class="section-label"><a href="/blog/">Blog</a> · <time datetime="{{.Post.Date.Format "2006-01-02"}}">{{.Post.Date.Format "2 January 2006"}}</time></p>
      <h1 class="blog-title">{{.Post.Title}}</h1>
      {{if .Post.Tags}}<p class="blog-tags">{{range .Post.Tags}}<span class="tag">{{.}}</span> {{end}}</p>{{end}}
      <div class="blog-body">{{.Body}}</div>
      <p class="blog-byline">— {{.Author}}</p>
    </article>
  </main>
` + shellBottom))

var indexTmpl = template.Must(template.New("index").Parse(shellTop + `
  <main class="blog-main">
    <section class="container">
      <p class="section-label">Blog</p>
      <h1 class="blog-title">Notes on building things that last.</h1>
      {{if .Posts}}
      <ul class="blog-list">
        {{range .Posts}}<li class="blog-item">
          <a href="/blog/{{.Slug}}/"><h2>{{.Title}}</h2></a>
          <p class="blog-meta"><time datetime="{{.Date.Format "2006-01-02"}}">{{.Date.Format "2 Jan 2006"}}</time></p>
          <p>{{.Teaser}}</p>
        </li>{{end}}
      </ul>
      {{else}}
      <p class="section-body">The first posts are on their way. Follow the <a href="/feed.xml">Atom feed</a> to get them.</p>
      {{end}}
    </section>
  </main>
` + shellBottom))

type indexItem struct {
	Post
	Teaser string
}

// BlogPostPage renders one post page (layout placeholders are filled by RenderLayout later).
func BlogPostPage(c Config, p Post) (string, error) {
	body, err := RenderMarkdown(p.Body)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	err = postTmpl.Execute(&b, map[string]any{"Lang": orDefault(c.Site.Language, "en"), "Title": p.Title + " — " + c.Site.Name,
		"Description": Teaser(p), "Post": p, "Body": body, "Author": c.Author.Name})
	return b.String(), err
}

// BlogIndexPage renders /blog/.
func BlogIndexPage(c Config, posts []Post) (string, error) {
	items := make([]indexItem, len(posts))
	for i, p := range posts {
		items[i] = indexItem{p, Teaser(p)}
	}
	var b strings.Builder
	err := indexTmpl.Execute(&b, map[string]any{"Lang": orDefault(c.Site.Language, "en"), "Title": "Blog — " + c.Site.Name,
		"Description": "Writing on the Nexovia Standard, delivery sovereignty and the tools built on it.", "Posts": items})
	return b.String(), err
}
