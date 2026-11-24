package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testDrafts() Drafts {
	return Drafts{Slug: "s", URL: "https://www.nex-ovia.com/blog/s/", LinkedIn: "Title\n\nBody", X: []string{"one", "two", "three"},
		Instagram: "cap", IGImage: "https://www.nex-ovia.com/og/s-ig.png"}
}

func TestLinkedInRequestShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/posts" || r.Header.Get("LinkedIn-Version") != "202510" || r.Header.Get("Authorization") != "Bearer tok" ||
			r.Header.Get("X-Restli-Protocol-Version") != "2.0.0" {
			t.Errorf("bad request: %s %v", r.URL.Path, r.Header)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		art := body["content"].(map[string]any)["article"].(map[string]any)
		if body["author"] != "urn:li:person:abc" || body["commentary"] != "Title\n\nBody" || art["source"] != "https://www.nex-ovia.com/blog/s/" || art["title"] != "Title" {
			t.Errorf("bad body: %v", body)
		}
		w.Header().Set("x-restli-id", "urn:li:share:1")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	id, err := LinkedIn{Base: srv.URL, Token: "tok", Author: "urn:li:person:abc", Version: "202510", HTTP: srv.Client()}.Publish(context.Background(), testDrafts())
	if err != nil || id != "urn:li:share:1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestXThreadRequests(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := atomic.AddInt32(&n, 1)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "OAuth ") || !strings.Contains(r.Header.Get("Authorization"), `oauth_signature="`) {
			t.Errorf("missing OAuth header")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if i > 1 {
			reply := body["reply"].(map[string]any)
			if reply["in_reply_to_tweet_id"] != "id"+string(rune('0'+i-1)) {
				t.Errorf("post %d should reply to previous: %v", i, body)
			}
		}
		_, _ = io.WriteString(w, `{"data":{"id":"id`+string(rune('0'+i))+`"}}`)
	}))
	defer srv.Close()
	id, err := X{Base: srv.URL, ConsumerKey: "ck", ConsumerSecret: "cs", Token: "t", TokenSec: "ts", HTTP: srv.Client()}.Publish(context.Background(), testDrafts())
	if err != nil || id != "id1" || n != 3 {
		t.Fatalf("id=%q n=%d err=%v", id, n, err)
	}
}

// Known-answer test from the X/Twitter OAuth 1.0a documentation ("Creating a signature").
func TestXOAuthSignatureKnownAnswer(t *testing.T) {
	x := X{ConsumerKey: "xvz1evFS4wEEPTGEFPHBog", ConsumerSecret: "kAcSOqF21Fu85e7zjz7ZN2U4ZRhfV3WpwPAoE3Z7kBw",
		Token: "370773112-GmHxMAgYyLbNEtIKZeRNFsMKPR9EyMZeS9weJAEb", TokenSec: "LswwdoUaIvS8ltyTt5jkRh4J50vUPVVHtR2YPi5kE",
		Now: func() time.Time { return time.Unix(1318622958, 0) }, Nonce: func() string { return "kYjzVBB8Y0ZFabxSWbWovY3uYSQ2pTgmZeNu2VS4cg" }}
	u, _ := url.Parse("https://api.twitter.com/1.1/statuses/update.json?include_entities=true&status=" +
		url.QueryEscape("Hello Ladies + Gentlemen, a signed OAuth request!"))
	h := x.authHeader(http.MethodPost, u)
	if !strings.Contains(h, `oauth_signature="hCtSmYh%2BiHYCEqBWrE7C7hYmtUk%3D"`) {
		t.Fatalf("signature mismatch: %s", h)
	}
}

func TestInstagramTwoStepPublish(t *testing.T) {
	var polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v23.0/42/media":
			_ = r.ParseForm()
			if r.Form.Get("image_url") != "https://www.nex-ovia.com/og/s-ig.png" || r.Form.Get("caption") != "cap" {
				t.Errorf("bad container form: %v", r.Form)
			}
			_, _ = io.WriteString(w, `{"id":"c1"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v23.0/c1":
			if atomic.AddInt32(&polls, 1) < 2 {
				_, _ = io.WriteString(w, `{"status_code":"IN_PROGRESS"}`)
				return
			}
			_, _ = io.WriteString(w, `{"status_code":"FINISHED"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v23.0/42/media_publish":
			_ = r.ParseForm()
			if r.Form.Get("creation_id") != "c1" {
				t.Errorf("bad publish form: %v", r.Form)
			}
			_, _ = io.WriteString(w, `{"id":"m1"}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	ig := Instagram{Base: srv.URL, Version: "v23.0", UserID: "42", Token: "tok", HTTP: srv.Client(), Sleep: func(time.Duration) {}}
	id, err := ig.Publish(context.Background(), testDrafts())
	if err != nil || id != "m1" || polls != 2 {
		t.Fatalf("id=%q polls=%d err=%v", id, polls, err)
	}
}

func TestInstagramStopsOnContainerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = io.WriteString(w, `{"id":"c1"}`)
			return
		}
		_, _ = io.WriteString(w, `{"status_code":"ERROR"}`)
	}))
	defer srv.Close()
	_, err := Instagram{Base: srv.URL, Version: "v23.0", UserID: "42", Token: "t", HTTP: srv.Client(), Sleep: func(time.Duration) {}}.Publish(context.Background(), testDrafts())
	if err == nil || !strings.Contains(err.Error(), "ERROR") {
		t.Fatalf("want container error, got %v", err)
	}
}

func TestHTTPErrorsSurfaceStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"title":"Too Many Requests"}`)
	}))
	defer srv.Close()
	_, err := X{Base: srv.URL, HTTP: srv.Client()}.Publish(context.Background(), testDrafts())
	if err == nil || !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("want 429 error, got %v", err)
	}
}
