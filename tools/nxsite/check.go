package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	h1Re        = regexp.MustCompile(`(?i)<h1[\s>]`)
	canonicalRe = regexp.MustCompile(`<link rel="canonical" href="[^"]+">`)
	descMetaRe  = regexp.MustCompile(`<meta name="description" content="([^"]+)">`)
	localRefRe  = regexp.MustCompile(`(?:href|src)="(/[^"#?]*)`)
)

// Check validates every HTML file under dist: exactly one <h1>, a canonical link, a non-empty
// description, and every root-relative href/src resolving to a built file. It returns all problems.
func Check(dist string) ([]string, error) {
	var problems []string
	err := filepath.WalkDir(dist, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		rel, _ := filepath.Rel(dist, path)
		if strings.HasPrefix(rel, "partials"+string(filepath.Separator)) {
			return nil // fragments kept for old cached clients only
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		problems = append(problems, checkPage(dist, rel, string(b))...)
		return nil
	})
	sort.Strings(problems)
	return problems, err
}

func checkPage(dist, rel, page string) []string {
	var out []string
	if n := len(h1Re.FindAllString(page, -1)); n != 1 {
		out = append(out, fmt.Sprintf("%s: %d <h1> elements, want 1", rel, n))
	}
	if rel != "404.html" && !canonicalRe.MatchString(page) {
		out = append(out, rel+": missing canonical link")
	}
	if !descMetaRe.MatchString(page) {
		out = append(out, rel+": missing or empty meta description")
	}
	seen := map[string]bool{}
	for _, m := range localRefRe.FindAllStringSubmatch(page, -1) {
		ref := m[1]
		if seen[ref] || strings.HasPrefix(ref, "//") || strings.HasPrefix(ref, "/cdn-cgi/") {
			continue
		}
		seen[ref] = true
		if !resolves(dist, ref) {
			out = append(out, fmt.Sprintf("%s: broken link %s", rel, ref))
		}
	}
	return out
}

func resolves(dist, ref string) bool {
	p := filepath.Join(dist, filepath.FromSlash(strings.TrimPrefix(ref, "/")))
	if strings.HasSuffix(ref, "/") {
		p = filepath.Join(p, "index.html")
	}
	st, err := os.Stat(p)
	if err == nil && st.IsDir() {
		_, err = os.Stat(filepath.Join(p, "index.html"))
	}
	return err == nil
}
