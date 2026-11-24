package main

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestInlineScriptHashesSkipsExternalAndData(t *testing.T) {
	page := `<script>console.log(1)</script><script src="/a.js"></script>` +
		`<script type="application/ld+json">{"a":1}</script><script type="module">x()</script><script>console.log(1)</script>`
	got := InlineScriptHashes([]string{page})
	sum := sha256.Sum256([]byte("console.log(1)"))
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if len(got) != 2 || !contains(got, want) {
		t.Fatalf("want 2 hashes incl. %s, got %v", want, got)
	}
}

func TestHeadersFileHasCSP(t *testing.T) {
	h := HeadersFile([]string{"'sha256-abc'"})
	for _, want := range []string{
		"Content-Security-Policy: default-src 'self'",
		"script-src 'self' 'sha256-abc' https://static.cloudflareinsights.com",
		"frame-ancestors 'none'",
		"Strict-Transport-Security: max-age=31536000",
		"X-Content-Type-Options: nosniff",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("missing %q", want)
		}
	}
	csp := CSP(nil)
	scriptSrc := strings.Split(strings.Split(csp, "script-src ")[1], ";")[0]
	if strings.Contains(scriptSrc, "unsafe") {
		t.Fatalf("script-src must not allow unsafe-*: %s", scriptSrc)
	}
}

func TestRedirectsAndNotFound(t *testing.T) {
	c := testConfig()
	c.Site.Name = `Ex <b>`
	nf := NotFoundPage(c)
	if !strings.Contains(nf, "<title>Page not found — Ex &lt;b&gt;</title>") || strings.Count(nf, "<h1") != 1 {
		t.Fatalf("404 page wrong:\n%s", nf)
	}
	if !strings.Contains(RedirectsFile(), "/sponsor/*  /connect/  301") || !strings.Contains(RedirectsFile(), "/work-with-me/*  /connect/  301") {
		t.Fatalf("sponsor redirect missing:\n%s", RedirectsFile())
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
