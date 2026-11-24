package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Beta reports whether a repo's latest release is pre-1.0, a prerelease, or missing.
func Beta(r Repo) bool {
	if r.Prerelease || r.Version == "" {
		return true
	}
	major, _, _ := strings.Cut(strings.TrimPrefix(r.Version, "v"), ".")
	return major == "0"
}

// LicenseLabel is the human label for an SPDX id.
func LicenseLabel(id string) string {
	switch id {
	case "":
		return "license not stated"
	case "BSL-1.1":
		return "BSL-1.1 (source-available)"
	case "MIT", "Apache-2.0", "BSD-3-Clause", "BSD-2-Clause", "MPL-2.0", "GPL-3.0", "AGPL-3.0":
		return id + " (open source)"
	}
	return id
}

// ToolCards renders one tool card per repo.
func ToolCards(repos []Repo) string {
	e := html.EscapeString
	var b strings.Builder
	for _, r := range repos {
		status := r.Version
		if r.Version == "" {
			status = "beta"
		} else if Beta(r) {
			status = "beta · " + r.Version
		}
		b.WriteString(`<div class="tool-card available reveal">
  <div class="tool-header">
    <div>
      <h2 class="tool-name">` + e(r.Name) + `</h2>
      <p class="tool-tagline">` + e(r.Description) + `</p>
    </div>
    <span class="status status-available">` + e(status) + `</span>
  </div>
  <div class="tool-body">
    <ul class="tool-features">
`)
		if r.Version != "" {
			b.WriteString(`      <li>Latest release: ` + e(r.Version) + ` · ` + e(r.Date) + `</li>
`)
		}
		b.WriteString(`      <li>License: ` + e(LicenseLabel(r.License)) + `</li>
      <li>GitHub stars: ` + fmt.Sprint(r.Stars) + `</li>
    </ul>
  </div>
  <div class="tool-footer">
    <span class="tool-footer-note">` + e(r.FullName) + `</span>
    <div class="tool-actions">
      <a href="` + e(r.URL) + `" class="btn btn-primary" rel="noopener noreferrer" target="_blank">★ Star on GitHub</a>
`)
		if h := r.Homepage; h != "" && h != r.URL && (strings.HasPrefix(h, "https://") || strings.HasPrefix(h, "http://")) {
			b.WriteString(`      <a href="` + e(h) + `" class="btn btn-secondary" rel="noopener noreferrer" target="_blank">Visit site →</a>
`)
		}
		b.WriteString(`    </div>
  </div>
</div>
`)
	}
	return b.String()
}

// ReplaceRegion replaces the content between <!-- github:name --> and <!-- /github:name -->, keeping both markers.
func ReplaceRegion(doc, name, body string) (string, bool) {
	open, end := "<!-- github:"+name+" -->", "<!-- /github:"+name+" -->"
	i := strings.Index(doc, open)
	if i < 0 {
		return doc, false
	}
	start := i + len(open)
	j := strings.Index(doc[start:], end)
	if j < 0 {
		return doc, false
	}
	return doc[:start] + "\n" + body + "\n" + doc[start+j:], true
}

// FillText sets the text of every element whose start tag carries attr (bare or with a value).
func FillText(doc, attr, value string) string {
	re := regexp.MustCompile(`(<[a-zA-Z0-9]+[^>]*\s` + regexp.QuoteMeta(attr) + `(?:[\s=/][^>]*)?>)[^<]*(</)`)
	v := html.EscapeString(value)
	return re.ReplaceAllStringFunc(doc, func(m string) string {
		g := re.FindStringSubmatch(m)
		return g[1] + v + g[2]
	})
}
