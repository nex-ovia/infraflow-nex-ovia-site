package main

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// fakeAPI serves canned results keyed by path.
type fakeAPI map[string]string

func (f fakeAPI) Get(path string, _ url.Values, out any) error {
	body, ok := f[path]
	if !ok {
		return errNotFound
	}
	return json.Unmarshal([]byte(body), out)
}

const (
	acct = "a0000000000000000000000000000000"
	zone = "z0000000000000000000000000000000"
)

func fixture() fakeAPI {
	return fakeAPI{
		"/zones/" + zone + "/rulesets": `[
			{"id":"r1","kind":"zone","phase":"http_request_dynamic_redirect"},
			{"id":"r2","kind":"managed","phase":"http_request_firewall_managed"},
			{"id":"r3","kind":"zone","phase":"http_ratelimit"}]`,
		"/accounts/" + acct + "/rum/site_info/list": `[
			{"site_tag":"other","ruleset":{"zone_tag":"elsewhere"}},
			{"site_tag":"s1","ruleset":{"zone_tag":"` + zone + `"}}]`,
		"/zones/" + zone + "/dns_records": `[
			{"id":"d1","type":"CNAME","name":"www.nex-ovia.com","content":"azure.example.net"},
			{"id":"d2","type":"CNAME","name":"nex-ovia.com","content":"azure.example.net"},
			{"id":"d3","type":"CNAME","name":"genome.nex-ovia.com","content":"nex-ovia.github.io"},
			{"id":"d4","type":"CAA","name":"nex-ovia.com","data":{"tag":"issue","value":"pki.goog"}},
			{"id":"d5","type":"CAA","name":"nex-ovia.com","data":{"tag":"issue","value":"evil-ca.example"}},
			{"id":"d6","type":"MX","name":"nex-ovia.com","content":"mx.example"}]`,
	}
}

func cfg(cutover bool) Config {
	return Config{Account: acct, Zone: zone, Apex: "nex-ovia.com", Host: "www.nex-ovia.com",
		Project: "nexovia-site", Cutover: cutover, CAAIssuers: []string{"letsencrypt.org", "pki.goog"}}
}

func addrs(imps []Import) map[string]string {
	m := map[string]string{}
	for _, i := range imps {
		m[i.To] = i.ID
	}
	return m
}

func TestCollectBeforeCutoverLeavesWWWAlone(t *testing.T) {
	imps, err := Collect(fixture(), cfg(false))
	if err != nil {
		t.Fatal(err)
	}
	got := addrs(imps)
	if _, ok := got["cloudflare_dns_record.www[0]"]; ok {
		t.Fatal("www must not be adopted before cutover")
	}
	want := map[string]string{
		"cloudflare_ruleset.redirects":                "zones/" + zone + "/r1",
		"cloudflare_ruleset.ratelimit":                "zones/" + zone + "/r3",
		"cloudflare_web_analytics_site.site":          acct + "/s1",
		`cloudflare_dns_record.caa_issue["pki.goog"]`: zone + "/d4",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d imports, want %d: %v", len(got), len(want), got)
	}
}

func TestCollectAtCutoverAdoptsWWWAndApexOnly(t *testing.T) {
	imps, err := Collect(fixture(), cfg(true))
	if err != nil {
		t.Fatal(err)
	}
	got := addrs(imps)
	if got["cloudflare_dns_record.www[0]"] != zone+"/d1" || got["cloudflare_dns_record.apex[0]"] != zone+"/d2" {
		t.Fatalf("www/apex not adopted: %v", got)
	}
	for k, v := range got {
		if strings.HasSuffix(v, "/d3") || strings.HasSuffix(v, "/d6") || strings.HasSuffix(v, "/d5") {
			t.Errorf("unmanaged record adopted as %s", k)
		}
	}
}

func TestCollectAdoptsExistingPagesProject(t *testing.T) {
	f := fixture()
	f["/accounts/"+acct+"/pages/projects/nexovia-site"] = `{"name":"nexovia-site"}`
	imps, err := Collect(f, cfg(false))
	if err != nil {
		t.Fatal(err)
	}
	if addrs(imps)["cloudflare_pages_project.site"] != acct+"/nexovia-site" {
		t.Fatal("existing Pages project not adopted")
	}
}

func TestRenderRejectsDuplicateTargets(t *testing.T) {
	var b bytes.Buffer
	err := Render(&b, []Import{{"x.y", "1"}, {"x.y", "2"}})
	if err == nil {
		t.Fatal("expected duplicate-target error")
	}
}

func TestRenderHCL(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, []Import{{`cloudflare_dns_record.caa_issue["pki.goog"]`, "z/d4"}}); err != nil {
		t.Fatal(err)
	}
	want := "import {\n  to = cloudflare_dns_record.caa_issue[\"pki.goog\"]\n  id = \"z/d4\"\n}\n"
	if !strings.Contains(b.String(), want) {
		t.Fatalf("unexpected HCL:\n%s", b.String())
	}
}
