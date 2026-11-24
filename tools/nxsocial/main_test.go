package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nex-ovia/infraflow-nex-ovia-site/tools/content"
)

type fakePub struct {
	name  string
	calls *int
	err   error
}

func (f fakePub) Name() string { return f.name }
func (f fakePub) Publish(context.Context, Drafts) (string, error) {
	*f.calls++
	return "id-" + f.name, f.err
}

func post(li, x, ig string) content.Post {
	p := samplePost()
	p.Social.LinkedIn, p.Social.X, p.Social.Instagram = li, x, ig
	return p
}

func TestDryRunMakesDraftsAndCallsNothing(t *testing.T) {
	calls := 0
	pubs := map[string]Publisher{"linkedin": fakePub{"linkedin", &calls, nil}, "x": fakePub{"x", &calls, nil}, "instagram": fakePub{"instagram", &calls, nil}}
	res := Run(context.Background(), post("auto", "auto", "auto"), testDrafts(), pubs, false)
	if calls != 0 {
		t.Fatalf("dry run made %d API calls", calls)
	}
	for _, r := range res {
		if r.Status != "draft" {
			t.Errorf("%s: %s, want draft", r.Platform, r.Status)
		}
	}
}

func TestRunPublishesOnlyAutoWithCredentials(t *testing.T) {
	calls := 0
	pubs := map[string]Publisher{"linkedin": fakePub{"linkedin", &calls, nil}, "x": fakePub{"x", &calls, errors.New("429")}}
	res := Run(context.Background(), post("auto", "auto", "auto"), testDrafts(), pubs, true)
	want := map[string]string{"linkedin": "published", "x": "failed", "instagram": "draft"}
	for _, r := range res {
		if r.Status != want[r.Platform] {
			t.Errorf("%s: %s, want %s", r.Platform, r.Status, want[r.Platform])
		}
	}
	if calls != 2 {
		t.Fatalf("want 2 calls, got %d", calls)
	}
	res = Run(context.Background(), post("off", "draft", "auto"), testDrafts(), map[string]Publisher{}, true)
	if res[0].Status != "skipped" || res[1].Status != "draft" || res[2].Err == nil {
		t.Fatalf("off/draft/no-creds handling wrong: %+v", res)
	}
}

func TestPublishersNeedAllCredentials(t *testing.T) {
	env := map[string]string{"LINKEDIN_TOKEN": "t", "LINKEDIN_AUTHOR_URN": "u", "X_API_KEY": "k", "IG_USER_ID": "1", "IG_ACCESS_TOKEN": "t"}
	pubs := Publishers(func(k string) string { return env[k] }, http.DefaultClient)
	if _, ok := pubs["x"]; ok {
		t.Error("x needs all four OAuth values")
	}
	if pubs["linkedin"].(LinkedIn).Version != "202510" || pubs["instagram"].(Instagram).Version != "v23.0" {
		t.Error("default API versions not applied")
	}
}

func TestWaitLive(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 3 {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	if err := WaitLive(context.Background(), srv.Client(), srv.URL, 5, time.Millisecond); err != nil || n != 3 {
		t.Fatalf("err=%v n=%d", err, n)
	}
	if err := WaitLive(context.Background(), srv.Client(), srv.URL+"/x", 1, time.Millisecond); err != nil {
		t.Fatalf("now live, got %v", err)
	}
}
