package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// inlineScriptRe captures inline <script> bodies; scripts with src= or a non-JS type are skipped.
var inlineScriptRe = regexp.MustCompile(`(?is)<script(\s[^>]*)?>(.*?)</script>`)

// InlineScriptHashes returns CSP 'sha256-…' sources for every executable inline script in pages.
func InlineScriptHashes(pages []string) []string {
	set := map[string]bool{}
	for _, p := range pages {
		for _, m := range inlineScriptRe.FindAllStringSubmatch(p, -1) {
			attrs := strings.ToLower(m[1])
			if strings.Contains(attrs, "src=") || (strings.Contains(attrs, "type=") && !strings.Contains(attrs, "javascript") && !strings.Contains(attrs, "module")) {
				continue
			}
			sum := sha256.Sum256([]byte(m[2]))
			set["'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'"] = true
		}
	}
	out := make([]string, 0, len(set))
	for h := range set {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

// CSP builds the Content-Security-Policy. Scripts: self + hashed inline scripts + Cloudflare Web
// Analytics. No 'unsafe-inline' or 'unsafe-eval' for scripts. Inline style attributes are allowed
// because the hand-written pages use them; styles can't execute code.
func CSP(scriptHashes []string) string {
	script := append([]string{"'self'"}, scriptHashes...)
	script = append(script, "https://static.cloudflareinsights.com")
	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + strings.Join(script, " "),
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data:",
		"connect-src 'self' https://cloudflareinsights.com",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'none'",
		"frame-ancestors 'none'",
		"upgrade-insecure-requests",
	}, "; ")
}

// HeadersFile renders the Cloudflare Pages _headers file.
func HeadersFile(scriptHashes []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "/*\n")
	for _, h := range [][2]string{
		{"Content-Security-Policy", CSP(scriptHashes)},
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()"},
		{"Cross-Origin-Opener-Policy", "same-origin"},
	} {
		fmt.Fprintf(&b, "  %s: %s\n", h[0], h[1])
	}
	// Long cache for static assets, short for HTML and feeds so new posts show quickly.
	b.WriteString("\n/og/*\n  Cache-Control: public, max-age=604800\n")
	b.WriteString("\n/*.css\n  Cache-Control: public, max-age=86400, stale-while-revalidate=604800\n")
	b.WriteString("\n/*.js\n  Cache-Control: public, max-age=86400, stale-while-revalidate=604800\n")
	b.WriteString("\n/feed.xml\n  Content-Type: application/atom+xml; charset=utf-8\n  Cache-Control: public, max-age=600\n")
	b.WriteString("\n/llms.txt\n  Content-Type: text/plain; charset=utf-8\n")
	b.WriteString("\n/project.toml\n  Content-Type: text/plain; charset=utf-8\n")
	return b.String()
}

// RedirectsFile renders the Cloudflare Pages _redirects file.
func RedirectsFile() string {
	return strings.Join([]string{
		"/rss.xml    /feed.xml  301",
		"/rss        /feed.xml  301",
		"/sponser    /connect/  301",
		"/sponser/*  /connect/  301",
		"/sponsor    /connect/  301",
		"/sponsor/*  /connect/  301",
		"/work-with-me    /connect/  301",
		"/work-with-me/*  /connect/  301",
		"/standard    /nxs/  301",
		"/standard/*  /nxs/  301",
		"/principles    /nxs/#principles  301",
		"/principles/*  /nxs/#principles  301",
		"/changelog    /nxs/  301",
		"/changelog/*  /nxs/  301",
	}, "\n") + "\n"
}

// StaticWebAppConfig renders RedirectsFile as Azure Static Web Apps routes (SWA ignores _redirects).
func StaticWebAppConfig() ([]byte, error) {
	type route struct {
		Route      string `json:"route"`
		Redirect   string `json:"redirect"`
		StatusCode int    `json:"statusCode"`
	}
	var routes []route
	for _, line := range strings.Split(strings.TrimSpace(RedirectsFile()), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			return nil, fmt.Errorf("bad redirect line %q", line)
		}
		code, err := strconv.Atoi(f[2])
		if err != nil {
			return nil, fmt.Errorf("bad redirect code in %q: %w", line, err)
		}
		routes = append(routes, route{f[0], f[1], code})
	}
	return json.MarshalIndent(map[string][]route{"routes": routes}, "", "  ")
}

// NotFoundPage is a branded 404 using the blog shell (layout and head are filled in by the build).
func NotFoundPage(c Config) string {
	return renderShell(c, "Page not found — "+c.Site.Name, "This page does not exist.", `
  <main class="blog-main">
    <section class="container">
      <p class="section-label">404</p>
      <h1 class="blog-title">This page doesn't exist.</h1>
      <p class="section-body">It may have moved. Try the <a href="/">home page</a>, the <a href="/tools/">tools</a> or the <a href="/about/">about page</a>.</p>
    </section>
  </main>
`)
}

// renderShell fills shellTop/shellBottom without html/template; main must be trusted static HTML.
func renderShell(c Config, title, desc, main string) string {
	esc := html.EscapeString
	top := strings.NewReplacer("{{.Lang}}", esc(orDefault(c.Site.Language, "en")), "{{.Title}}", esc(title), "{{.Description}}", esc(desc)).Replace(shellTop)
	return top + main + shellBottom
}
