package main

import (
	"strings"
	"testing"
)

var testRoutes = map[string]bool{"/": true, "/tools/": true, "/blog/": true}

func TestRenderLayoutPutsNavInRawHTML(t *testing.T) {
	page := `<body><div id="nav-placeholder"></div><h1>x</h1><div id="footer-placeholder"></div></body>`
	nav := `<nav><ul><li><a href="/tools">Tools</a></li><li><a href="/blog">B</a></li></ul></nav>`
	out := RenderLayout(page, nav, `<footer><a href="/blog">B</a></footer>`, "/tools/", testRoutes)
	for _, want := range []string{
		`<li class="active"><a href="/tools/">Tools</a>`,
		`<li><a href="/blog/">B</a></li>`,
		`<footer><a href="/blog/">B</a></footer>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

func TestNormalizeLinksKeepsAnchorsAndUnknownPaths(t *testing.T) {
	in := `<a href="/tools#x"><a href="/site.css"><a href="https://x.com/tools"><a href="/">`
	out := normalizeLinks(in, testRoutes)
	want := `<a href="/tools/#x"><a href="/site.css"><a href="https://x.com/tools"><a href="/">`
	if out != want {
		t.Fatalf("got  %s\nwant %s", out, want)
	}
}

func TestMarkActiveAppendsToExistingClass(t *testing.T) {
	out := markActive(`<li class="highlight"><a href="/blog/">B</a></li>`, "/blog/")
	if !strings.Contains(out, `class="highlight active"`) {
		t.Fatal(out)
	}
	if markActive(`<li><a href="/">H</a></li>`, "/") != `<li><a href="/">H</a></li>` {
		t.Fatal("home link should not be marked active")
	}
}

func TestAddLinkAfter(t *testing.T) {
	in := `<ul><li><a href="/tools">Tools</a></li><li><a href="/changelog">C</a></li></ul>`
	out := AddLinkAfter(in, "/tools", "/blog", "Blog")
	if out != `<ul><li><a href="/tools">Tools</a></li><li><a href="/blog">Blog</a></li><li><a href="/changelog">C</a></li></ul>` {
		t.Fatal(out)
	}
	if AddLinkAfter(out, "/tools", "/blog", "Blog") != out {
		t.Fatal("must not add a duplicate")
	}
	if AddLinkAfter(in, "/missing", "/blog", "Blog") != in {
		t.Fatal("unknown anchor must leave fragment unchanged")
	}
}
