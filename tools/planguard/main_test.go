package main

import (
	"strings"
	"testing"
)

const planJSON = `{"resource_changes":[
 {"address":"cloudflare_dns_record.apex[0]","type":"cloudflare_dns_record","change":{"actions":["delete","create"]}},
 {"address":"cloudflare_dns_record.caa_iodef","type":"cloudflare_dns_record","change":{"actions":["delete"]}},
 {"address":"cloudflare_pages_project.site","type":"cloudflare_pages_project","change":{"actions":["update"]}},
 {"address":"cloudflare_ruleset.firewall","type":"cloudflare_ruleset","change":{"actions":["delete"]}}
]}`

func TestViolationsFlagsProtectedDeletes(t *testing.T) {
	v, err := Violations(strings.NewReader(planJSON), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 2 {
		t.Fatalf("want 2 violations (apex replace, caa delete), got %v", v)
	}
}

func TestViolationsHonoursAllowList(t *testing.T) {
	v, err := Violations(strings.NewReader(planJSON), map[string]bool{"cloudflare_dns_record.apex[0]": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 || !strings.HasPrefix(v[0], "cloudflare_dns_record.caa_iodef") {
		t.Fatalf("want only caa_iodef, got %v", v)
	}
}

func TestViolationsIgnoresUnprotectedTypes(t *testing.T) {
	v, err := Violations(strings.NewReader(`{"resource_changes":[{"address":"cloudflare_ruleset.x","type":"cloudflare_ruleset","change":{"actions":["delete"]}}]}`), nil)
	if err != nil || len(v) != 0 {
		t.Fatalf("ruleset delete should pass, got %v %v", v, err)
	}
}

func TestViolationsRejectsBadJSON(t *testing.T) {
	if _, err := Violations(strings.NewReader("not json"), nil); err == nil {
		t.Fatal("expected error")
	}
}
