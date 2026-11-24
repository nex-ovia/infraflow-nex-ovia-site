package main

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/nex-ovia/infraflow-nex-ovia-site/tools/content"
)

func samplePost() content.Post {
	return content.Post{Slug: "build-once", Title: "Build it once, own it forever",
		Date: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC), Tags: []string{"nxs", "delivery-sovereignty", "nxs", "c++"},
		Summary: strings.Repeat("Delivery sovereignty means the team keeps understanding and control. ", 12)}
}

func TestDraftLimits(t *testing.T) {
	d := MakeDrafts(samplePost(), "https://www.nex-ovia.com/")
	if d.URL != "https://www.nex-ovia.com/blog/build-once/" || d.IGImage != "https://www.nex-ovia.com/og/build-once-ig.png" {
		t.Fatalf("urls wrong: %+v", d)
	}
	if n := utf8.RuneCountInString(d.LinkedIn); n > linkedInMax || !strings.Contains(d.LinkedIn, d.URL) {
		t.Errorf("linkedin bad (%d chars)", n)
	}
	if len(d.X) < 2 || !strings.Contains(d.X[0], d.URL) {
		t.Fatalf("x thread should start with the link and continue: %q", d.X)
	}
	for i, p := range d.X {
		if XLen(p) > xMax {
			t.Errorf("x post %d is %d chars", i, XLen(p))
		}
	}
	if n := utf8.RuneCountInString(d.Instagram); n > instagramMax || strings.Contains(d.Instagram, "https://") {
		t.Errorf("instagram caption bad (%d chars, links not clickable there)", n)
	}
	if !strings.Contains(d.Instagram, "#nxs #deliverysovereignty #c") || strings.Count(d.Instagram, "#nxs ") != 1 {
		t.Errorf("hashtags not normalised/deduped: %s", d.Instagram)
	}
}

func TestXLenWeighsURLs(t *testing.T) {
	if got := XLen("hi https://example.com/very/long/path/that/is/long"); got != 3+23 {
		t.Fatalf("XLen = %d, want 26", got)
	}
}

func TestClipXShortensLongTitle(t *testing.T) {
	s := clipX(strings.Repeat("long title ", 40) + "\n\nhttps://www.nex-ovia.com/blog/x/")
	if XLen(s) > xMax || !strings.HasSuffix(s, "https://www.nex-ovia.com/blog/x/") {
		t.Fatalf("clipX failed (%d): %q", XLen(s), s)
	}
}

func TestRenderListsAllPlatforms(t *testing.T) {
	out := MakeDrafts(samplePost(), "https://www.nex-ovia.com").Render()
	for _, want := range []string{"## LinkedIn", "## X thread", "## Instagram", "image: https://www.nex-ovia.com/og/build-once-ig.png"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
}
