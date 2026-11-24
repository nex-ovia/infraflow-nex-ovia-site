package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFillMetrics(t *testing.T) {
	html := `<td data-metric="a/b:releases">1</td><td data-metric="a/b:latest">old</td><td data-metric="c/d:downloads">7</td>`
	fetch := func(r string) (RepoStats, error) {
		if r == "c/d" {
			return RepoStats{}, errors.New("boom")
		}
		return RepoStats{Releases: 9, Latest: "v1 · 2026-01-02"}, nil
	}
	got := FillMetrics(html, fetch)
	want := `<td data-metric="a/b:releases">9</td><td data-metric="a/b:latest">v1 · 2026-01-02</td><td data-metric="c/d:downloads">7</td>`
	if got != want {
		t.Fatalf("got %s", got)
	}
	if d := FillMetricDate(`<span data-metric-date>x</span>`, "2026-12-01"); d != `<span data-metric-date>2026-12-01</span>` {
		t.Fatalf("date %s", d)
	}
}

func TestGitHubStats(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("Authorization") != "Bearer tok" || !strings.HasSuffix(r.URL.Path, "/repos/o/r/releases") {
			http.Error(w, "bad", 400)
			return
		}
		w.Write([]byte(`[{"tag_name":"v2","published_at":"2026-03-04T10:00:00Z","assets":[{"download_count":5},{"download_count":2}]},{"tag_name":"v1","published_at":"2026-01-01T00:00:00Z","assets":[{"download_count":3}]}]`))
	}))
	defer srv.Close()
	f := githubStats(srv.Client(), "tok", srv.URL)
	for i := 0; i < 2; i++ {
		st, err := f("o/r")
		if err != nil || st != (RepoStats{2, 10, "v2 · 2026-03-04"}) {
			t.Fatalf("%v %v", st, err)
		}
	}
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}
}
