// Command cfimport writes infra/import.tf: Terraform import blocks for Cloudflare objects that
// already exist, so `terraform apply` adopts them instead of failing or recreating them.
//
//	CLOUDFLARE_API_TOKEN=... cfimport -account <id> -zone <id> -apex nex-ovia.com -host www.nex-ovia.com \
//	    -project nexovia-site [-cutover] > infra/import.tf
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

// API is the subset of the Cloudflare v4 API cfimport reads.
type API interface {
	Get(path string, query url.Values, out any) error
}

type httpAPI struct {
	base  string
	token string
	c     *http.Client
}

func (h httpAPI) Get(path string, q url.Values, out any) error {
	u := h.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	resp, err := h.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: HTTP %d", path, resp.StatusCode)
	}
	var env struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	if !env.Success {
		return fmt.Errorf("GET %s: success=false", path)
	}
	return json.Unmarshal(env.Result, out)
}

var errNotFound = fmt.Errorf("not found")

// Config names the objects this stack manages.
type Config struct {
	Account, Zone, Apex, Host, Project string
	Cutover                            bool
	CAAIssuers                         []string
}

// Import is one Terraform import block.
type Import struct{ To, ID string }

type dnsRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Data    struct {
		Tag   string `json:"tag"`
		Value string `json:"value"`
	} `json:"data"`
}

type ruleset struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Phase string `json:"phase"`
}

type rumSite struct {
	SiteTag string `json:"site_tag"`
	Ruleset struct {
		ZoneTag string `json:"zone_tag"`
	} `json:"ruleset"`
}

// phaseResource maps ruleset phases to the resource that owns them (one zone ruleset per phase).
var phaseResource = map[string]string{
	"http_request_dynamic_redirect": "cloudflare_ruleset.redirects",
	"http_request_firewall_custom":  "cloudflare_ruleset.firewall",
	"http_ratelimit":                "cloudflare_ruleset.ratelimit",
}

// Collect queries the API and returns import blocks in a stable order.
func Collect(api API, c Config) ([]Import, error) {
	var out []Import
	steps := []func(API, Config) ([]Import, error){collectPages, collectRulesets, collectAnalytics, collectDNS}
	for _, step := range steps {
		got, err := step(api, c)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].To < out[j].To })
	return out, nil
}

func collectPages(api API, c Config) ([]Import, error) {
	var p struct {
		Name string `json:"name"`
	}
	err := api.Get(fmt.Sprintf("/accounts/%s/pages/projects/%s", c.Account, c.Project), nil, &p)
	if err == errNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []Import{{"cloudflare_pages_project.site", c.Account + "/" + c.Project}}, nil
}

func collectRulesets(api API, c Config) ([]Import, error) {
	var rs []ruleset
	if err := api.Get("/zones/"+c.Zone+"/rulesets", nil, &rs); err != nil {
		return nil, err
	}
	var out []Import
	for _, r := range rs {
		if to, ok := phaseResource[r.Phase]; ok && r.Kind == "zone" {
			out = append(out, Import{to, "zones/" + c.Zone + "/" + r.ID})
		}
	}
	return out, nil
}

func collectAnalytics(api API, c Config) ([]Import, error) {
	var sites []rumSite
	q := url.Values{"per_page": {"50"}}
	if err := api.Get("/accounts/"+c.Account+"/rum/site_info/list", q, &sites); err != nil {
		return nil, err
	}
	for _, s := range sites {
		if s.Ruleset.ZoneTag == c.Zone {
			return []Import{{"cloudflare_web_analytics_site.site", c.Account + "/" + s.SiteTag}}, nil
		}
	}
	return nil, nil
}

func collectDNS(api API, c Config) ([]Import, error) {
	var recs []dnsRecord
	q := url.Values{"per_page": {"500"}}
	if err := api.Get("/zones/"+c.Zone+"/dns_records", q, &recs); err != nil {
		return nil, err
	}
	issuers := map[string]bool{}
	for _, i := range c.CAAIssuers {
		issuers[i] = true
	}
	var out []Import
	for _, r := range recs {
		if to := dnsTarget(r, c, issuers); to != "" {
			out = append(out, Import{to, c.Zone + "/" + r.ID})
		}
	}
	return out, nil
}

// dnsTarget returns the resource address a record maps to, or "" when the stack doesn't manage it.
func dnsTarget(r dnsRecord, c Config, issuers map[string]bool) string {
	name := strings.TrimSuffix(r.Name, ".")
	switch {
	case r.Type == "CAA" && name == c.Apex && r.Data.Tag == "issue" && issuers[r.Data.Value]:
		return fmt.Sprintf("cloudflare_dns_record.caa_issue[%q]", r.Data.Value)
	case r.Type == "CAA" && name == c.Apex && r.Data.Tag == "iodef":
		return "cloudflare_dns_record.caa_iodef"
	case !c.Cutover:
		return ""
	case name == c.Host && (r.Type == "CNAME" || r.Type == "A" || r.Type == "AAAA"):
		return "cloudflare_dns_record.www[0]"
	case name == c.Apex && (r.Type == "CNAME" || r.Type == "A" || r.Type == "AAAA"):
		return "cloudflare_dns_record.apex[0]"
	}
	return ""
}

// Render writes the import blocks as HCL.
func Render(w io.Writer, imps []Import) error {
	if _, err := fmt.Fprintln(w, "# Generated by tools/cfimport — do not edit. Adopts existing Cloudflare objects."); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, i := range imps {
		if seen[i.To] {
			return fmt.Errorf("two existing objects map to %s; resolve by hand", i.To)
		}
		seen[i.To] = true
		if _, err := fmt.Fprintf(w, "\nimport {\n  to = %s\n  id = %q\n}\n", i.To, i.ID); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var c Config
	var caa string
	flag.StringVar(&c.Account, "account", "", "Cloudflare account ID")
	flag.StringVar(&c.Zone, "zone", "", "Cloudflare zone ID")
	flag.StringVar(&c.Apex, "apex", "nex-ovia.com", "apex domain")
	flag.StringVar(&c.Host, "host", "www.nex-ovia.com", "site host")
	flag.StringVar(&c.Project, "project", "nexovia-site", "Pages project name")
	flag.BoolVar(&c.Cutover, "cutover", false, "also adopt www/apex records")
	flag.StringVar(&caa, "caa", "letsencrypt.org,pki.goog,ssl.com,digicert.com", "CAA issuers managed by the stack")
	flag.Parse()
	c.CAAIssuers = strings.Split(caa, ",")
	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	if token == "" || c.Account == "" || c.Zone == "" {
		fmt.Fprintln(os.Stderr, "cfimport: need CLOUDFLARE_API_TOKEN, -account and -zone")
		os.Exit(2)
	}
	api := httpAPI{base: "https://api.cloudflare.com/client/v4", token: token, c: http.DefaultClient}
	imps, err := Collect(api, c)
	if err == nil {
		err = Render(os.Stdout, imps)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cfimport:", err)
		os.Exit(1)
	}
}
