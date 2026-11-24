package main

import (
	"regexp"
	"strings"
)

var (
	navPlaceholderRe    = regexp.MustCompile(`<div id="nav-placeholder">\s*</div>`)
	footerPlaceholderRe = regexp.MustCompile(`<div id="footer-placeholder">\s*</div>`)
	internalHrefRe      = regexp.MustCompile(`href="(/[^"#?]*)([#?][^"]*)?"`)
	navItemRe           = regexp.MustCompile(`<li(\s+class="([^"]*)")?>(\s*<a href="([^"]+)")`)
	mainOpenRe          = regexp.MustCompile(`<main(\s[^>]*)?>`)
	idAttrRe            = regexp.MustCompile(`\sid="([^"]*)"`)
	bodyOpenRe          = regexp.MustCompile(`<body[^>]*>`)
)

// RenderLayout server-renders nav/footer into a page (so crawlers see the links without JS),
// marks the current nav item active and rewrites internal links to canonical trailing-slash routes.
func RenderLayout(page, nav, footer, route string, routes map[string]bool) string {
	nav = markActive(normalizeLinks(nav, routes), route)
	footer = normalizeLinks(footer, routes)
	m := mainOpenRe.FindStringIndex(page)
	if m == nil {
		// No main landmark: wrap exactly the content between nav and footer.
		if navPlaceholderRe.MatchString(page) {
			nav += "\n<main id=\"main\">"
		} else if i := strings.Index(page, "</nav>"); i >= 0 { // page carries its own inline nav
			page = page[:i+len("</nav>")] + "\n<main id=\"main\">" + page[i+len("</nav>"):]
		} else if b := bodyOpenRe.FindStringIndex(page); b != nil {
			page = page[:b[1]] + "\n<main id=\"main\">" + page[b[1]:]
		}
		if footerPlaceholderRe.MatchString(page) {
			footer = "</main>\n" + footer
		} else {
			page = strings.Replace(page, "</body>", "</main>\n</body>", 1)
		}
	} else if id := idAttrRe.FindStringSubmatch(page[m[0]:m[1]]); id == nil {
		page = page[:m[0]] + `<main id="main"` + page[m[0]+len("<main"):]
	} else if id[1] != "main" {
		nav = strings.Replace(nav, `href="#main"`, `href="#`+id[1]+`"`, 1)
	}
	out := navPlaceholderRe.ReplaceAllLiteralString(page, nav)
	out = footerPlaceholderRe.ReplaceAllLiteralString(out, footer)
	return normalizeLinks(out, routes)
}

// normalizeLinks rewrites href="/x" to "/x/" when /x/ is a known route.
func normalizeLinks(s string, routes map[string]bool) string {
	return internalHrefRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := internalHrefRe.FindStringSubmatch(m)
		path, suffix := sub[1], sub[2]
		if !strings.HasSuffix(path, "/") && routes[path+"/"] {
			path += "/"
		}
		return `href="` + path + suffix + `"`
	})
}

// markActive adds class "active" to the <li> whose link equals route and aria-current="page" to its <a>.
func markActive(nav, route string) string {
	return navItemRe.ReplaceAllStringFunc(nav, func(m string) string {
		sub := navItemRe.FindStringSubmatch(m)
		classes, rest, href := sub[2], sub[3], sub[4]
		if href != route || route == "/" {
			return m
		}
		rest += ` aria-current="page"`
		if classes == "" {
			return `<li class="active">` + rest
		}
		return `<li class="` + classes + ` active">` + rest
	})
}

// AddLinkAfter inserts <li><a href="href">label</a></li> after the <li> linking to after, unless a
// link to href already exists. Used to add build-only nav items (e.g. Blog) without changing the
// partials the legacy host still serves.
func AddLinkAfter(fragment, after, href, label string) string {
	if strings.Contains(fragment, `href="`+href+`"`) || strings.Contains(fragment, `href="`+href+`/"`) {
		return fragment
	}
	marker := `<a href="` + after + `">`
	i := strings.Index(fragment, marker)
	if i < 0 {
		return fragment
	}
	end := strings.Index(fragment[i:], "</li>")
	if end < 0 {
		return fragment
	}
	cut := i + end + len("</li>")
	return fragment[:cut] + `<li><a href="` + href + `">` + label + `</a></li>` + fragment[cut:]
}
