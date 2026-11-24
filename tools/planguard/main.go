// Command planguard reads `terraform show -json <plan>` on stdin and fails when the plan would delete
// or replace a protected resource. Protected: every DNS record, the Pages project, DNSSEC.
// Replacements listed in -allow (comma-separated addresses) are permitted, e.g. the www/apex records
// that change type at cutover.
//
//	terraform show -json tfplan | planguard -allow 'cloudflare_dns_record.apex[0]'
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type plan struct {
	ResourceChanges []struct {
		Address string `json:"address"`
		Type    string `json:"type"`
		Change  struct {
			Actions []string `json:"actions"`
		} `json:"change"`
	} `json:"resource_changes"`
}

var protectedTypes = map[string]bool{
	"cloudflare_dns_record":    true,
	"cloudflare_pages_project": true,
	"cloudflare_zone_dnssec":   true,
}

// Violations lists protected resources the plan would delete, with a reason line each.
func Violations(r io.Reader, allow map[string]bool) ([]string, error) {
	var p plan
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, fmt.Errorf("reading plan JSON: %w", err)
	}
	var out []string
	for _, rc := range p.ResourceChanges {
		if !protectedTypes[rc.Type] || allow[rc.Address] {
			continue
		}
		for _, a := range rc.Change.Actions {
			if a == "delete" {
				out = append(out, fmt.Sprintf("%s: %s", rc.Address, strings.Join(rc.Change.Actions, "+")))
				break
			}
		}
	}
	return out, nil
}

func main() {
	allowFlag := flag.String("allow", "", "comma-separated resource addresses that may be replaced")
	flag.Parse()
	allow := map[string]bool{}
	for _, a := range strings.Split(*allowFlag, ",") {
		if a = strings.TrimSpace(a); a != "" {
			allow[a] = true
		}
	}
	v, err := Violations(os.Stdin, allow)
	if err != nil {
		fmt.Fprintln(os.Stderr, "planguard:", err)
		os.Exit(2)
	}
	if len(v) > 0 {
		fmt.Fprintln(os.Stderr, "planguard: plan deletes protected resources:")
		for _, l := range v {
			fmt.Fprintln(os.Stderr, "  -", l)
		}
		os.Exit(1)
	}
	fmt.Println("planguard: ok, no protected resource is deleted")
}
