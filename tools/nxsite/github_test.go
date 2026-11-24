package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func fakeGitHub(t *testing.T) GitHub {
	t.Helper()
	mux := http.NewServeMux()
	serve := func(path, body string) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Accept") != "application/vnd.github+json" || r.Header.Get("Authorization") != "Bearer tok" {
				http.Error(w, "bad headers", http.StatusBadRequest)
				return
			}
			w.Write([]byte(body))
		})
	}
	bsl := base64.StdEncoding.EncodeToString([]byte("Business Source License 1.1"))
	bsl = bsl[:8] + `\n` + bsl[8:] // GitHub wraps content with newlines
	mux.HandleFunc("/search/repositories", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "q=topic%3Anxs-tool+user%3Aa+user%3Ab&sort=stars&order=desc&per_page=100" {
			t.Errorf("search query = %q", got)
		}
		w.Write([]byte(`{"items":[
 {"name":"genome","full_name":"a/genome","description":"Graph","homepage":"https://g.example","html_url":"https://github.com/a/genome","stargazers_count":12,"archived":false,"fork":false,"license":{"spdx_id":"NOASSERTION"}},
 {"name":"old","full_name":"a/old","archived":true,"fork":false,"license":{"spdx_id":"MIT"}},
 {"name":"lib","full_name":"b/lib","description":"Lib","homepage":"","html_url":"https://github.com/b/lib","stargazers_count":3,"archived":false,"fork":false,"license":{"spdx_id":"MIT"}}
]}`))
	})
	serve("/repos/a/genome/releases", `[{"tag_name":"v0.1.0-alpha.1","published_at":"2026-10-01T12:00:00Z","prerelease":true}]`)
	serve("/repos/a/genome/license", `{"content":"`+bsl+`"}`)
	serve("/repos/b/lib/releases", `[]`)
	serve("/repos/a/coderev/tags", `[{"name":"v0.1.0","commit":{"sha":"c1"}},{"name":"v0.2.0","commit":{"sha":"c2"}}]`)
	serve("/repos/a/coderev/git/ref/tags/v0.1.0", `{"object":{"type":"commit","sha":"c1"}}`)
	serve("/repos/a/coderev/commits/c1", `{"commit":{"message":"first cut\n\nbody","committer":{"date":"2026-05-10T08:00:00Z"}}}`)
	serve("/repos/a/coderev/git/ref/tags/v0.2.0", `{"object":{"type":"tag","sha":"t2"}}`)
	serve("/repos/a/coderev/git/tags/t2", `{"message":"  second release \nmore","tagger":{"date":"2026-09-02T10:00:00Z"}}`)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return GitHub{Client: srv.Client(), Token: "tok", API: srv.URL}
}

func TestToolRepos(t *testing.T) {
	got, err := fakeGitHub(t).ToolRepos([]string{"a", "b"}, "nxs-tool")
	if err != nil {
		t.Fatal(err)
	}
	want := []Repo{
		{Name: "genome", FullName: "a/genome", Description: "Graph", Homepage: "https://g.example",
			URL: "https://github.com/a/genome", License: "BSL-1.1", Stars: 12,
			Version: "v0.1.0-alpha.1", Date: "2026-10-01", Prerelease: true},
		{Name: "lib", FullName: "b/lib", Description: "Lib", URL: "https://github.com/b/lib", License: "MIT", Stars: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToolRepos =\n%+v\nwant\n%+v", got, want)
	}
}

func TestVersions(t *testing.T) {
	got, err := fakeGitHub(t).Versions("a/coderev")
	if err != nil {
		t.Fatal(err)
	}
	want := []Version{
		{Tag: "v0.2.0", Date: "2026-09-02", Title: "second release"},
		{Tag: "v0.1.0", Date: "2026-05-10", Title: "first cut"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Versions = %+v, want %+v", got, want)
	}
}

func TestGetNon200(t *testing.T) {
	if _, err := fakeGitHub(t).Versions("a/missing"); err == nil {
		t.Fatal("want error for 404")
	}
}
