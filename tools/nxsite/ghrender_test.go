package main

import (
	"strings"
	"testing"
)

func TestBeta(t *testing.T) {
	for _, c := range []struct {
		r    Repo
		want bool
	}{
		{Repo{Version: "v0.30.4"}, true},
		{Repo{Version: "v1.2.0"}, false},
		{Repo{Version: "v1.0.0-rc1", Prerelease: true}, true},
		{Repo{Version: ""}, true},
	} {
		if got := Beta(c.r); got != c.want {
			t.Errorf("Beta(%q) = %v", c.r.Version, got)
		}
	}
}

func TestLicenseLabel(t *testing.T) {
	for in, want := range map[string]string{
		"":         "license not stated",
		"BSL-1.1":  "BSL-1.1 (source-available)",
		"MIT":      "MIT (open source)",
		"AGPL-3.0": "AGPL-3.0 (open source)",
		"Other":    "Other",
	} {
		if got := LicenseLabel(in); got != want {
			t.Errorf("LicenseLabel(%q) = %q", in, got)
		}
	}
}

func TestToolCards(t *testing.T) {
	for _, c := range []struct {
		r          Repo
		status     string
		visit, rel bool
	}{
		{Repo{Name: "<b>", URL: "https://github.com/o/a", Homepage: "https://a.dev", Version: "v0.1.0", Date: "2026-01-01"}, "beta · v0.1.0", true, true},
		{Repo{Name: "x", URL: "https://github.com/o/a", Homepage: "https://github.com/o/a", Version: "v1.0.0"}, "v1.0.0", false, true},
		{Repo{Name: "x", URL: "https://github.com/o/a", Homepage: "javascript:alert(1)"}, "beta", false, false},
		{Repo{Name: "x", URL: "https://github.com/o/a"}, "beta", false, false},
	} {
		out := ToolCards([]Repo{c.r})
		if strings.Contains(out, "<b>") {
			t.Errorf("unescaped name: %s", out)
		}
		if !strings.Contains(out, `<span class="status status-available">`+c.status+`</span>`) {
			t.Errorf("status %q missing: %s", c.status, out)
		}
		if strings.Contains(out, "Visit site") != c.visit {
			t.Errorf("visit link for %q: %s", c.r.Homepage, out)
		}
		if strings.Contains(out, "Latest release:") != c.rel {
			t.Errorf("release li: %s", out)
		}
	}
	if !strings.Contains(ToolCards([]Repo{{Name: "<b>"}}), "&lt;b&gt;") {
		t.Error("name not escaped")
	}
}

func TestVersionList(t *testing.T) {
	out := VersionList([]Version{{Tag: "v2", Date: "d2", Title: "<t>"}, {Tag: "v1 x", Date: "d1"}}, "o/r")
	if strings.Count(out, "Current") != 1 || strings.Index(out, "Current") > strings.Index(out, "v1 x") {
		t.Errorf("Current: %s", out)
	}
	if !strings.Contains(out, `href="https://github.com/o/r/releases/tag/v1%20x"`) || !strings.Contains(out, "&lt;t&gt;") {
		t.Errorf("href/escape: %s", out)
	}
}

func TestReplaceRegion(t *testing.T) {
	in := "a<!-- github:tools -->old<!-- /github:tools -->b"
	out, ok := ReplaceRegion(in, "tools", "NEW")
	if !ok || out != "a<!-- github:tools -->\nNEW\n<!-- /github:tools -->b" {
		t.Fatalf("%v %q", ok, out)
	}
	if out, ok := ReplaceRegion("a<!-- github:tools -->old", "tools", "NEW"); ok || out != "a<!-- github:tools -->old" {
		t.Fatalf("missing end: %v %q", ok, out)
	}
	if _, ok := ReplaceRegion(in, "other", "NEW"); ok {
		t.Fatal("missing start")
	}
}

func TestFillText(t *testing.T) {
	in := `<span data-x>a</span><b class="k" data-x="1">b</b><i data-xy>c</i>`
	want := `<span data-x>&lt;v&gt;</span><b class="k" data-x="1">&lt;v&gt;</b><i data-xy>c</i>`
	if got := FillText(in, "data-x", "<v>"); got != want {
		t.Fatalf("%s", got)
	}
}
