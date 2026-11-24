package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildFillsGitHubRegions(t *testing.T) {
	mux := http.NewServeMux()
	serve := func(path, body string) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
	}
	serve("/search/repositories", `{"items":[{"name":"fakegenome","full_name":"nex-ovia/fakegenome","description":"Fake graph","html_url":"https://github.com/nex-ovia/fakegenome","stargazers_count":5,"license":{"spdx_id":"MIT"}}]}`)
	serve("/repos/nex-ovia/fakegenome/releases", `[]`)
	serve("/repos/nex-ovia/NXS/tags", `[{"name":"v9.8.7","commit":{"sha":"c1"}}]`)
	serve("/repos/nex-ovia/NXS/git/ref/tags/v9.8.7", `{"object":{"type":"commit","sha":"c1"}}`)
	serve("/repos/nex-ovia/NXS/commits/c1", `{"commit":{"message":"fake release","committer":{"date":"2026-10-01T00:00:00Z"}}}`)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("NXSITE_GITHUB_API", srv.URL)

	page := `<!DOCTYPE html><html><head><title>Tools</title><meta name="description" content="Tools desc"></head>
<body><div id="nav-placeholder"></div><h1>Tools</h1><span data-nxs-version>vX</span>
<!-- github:tools -->FALLBACK-TOOLS<!-- /github:tools -->
<div id="footer-placeholder"></div></body></html>`
	mutate := func(root string) {
		if err := os.WriteFile(filepath.Join(root, "tools", "index.html"), []byte(page), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("NXSITE_OFFLINE", "0")
	_, out := buildFixtureWith(t, mutate)
	got := read(t, filepath.Join(out, "tools/index.html"))
	for _, want := range []string{"fakegenome", "Fake graph", "<span data-nxs-version>v9.8.7</span>"} {
		if !strings.Contains(got, want) {
			t.Errorf("online build missing %q", want)
		}
	}
	if strings.Contains(got, "FALLBACK-") {
		t.Error("online build kept a fallback")
	}

	t.Setenv("NXSITE_OFFLINE", "1")
	_, out = buildFixtureWith(t, mutate)
	got = read(t, filepath.Join(out, "tools/index.html"))
	for _, want := range []string{"FALLBACK-TOOLS","<span data-nxs-version>vX</span>"} {
		if !strings.Contains(got, want) {
			t.Errorf("offline build missing fallback %q", want)
		}
	}
}
