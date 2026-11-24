package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Repo is one tool repository found by topic.
type Repo struct {
	Name, FullName, Description, Homepage, URL, License string
	Stars                                               int
	Version                                             string // tag of the newest release, prereleases included; "" if none
	Date                                                string // YYYY-MM-DD of that release
	Prerelease                                          bool
}

// Version is one git tag. Title is the first line of the tag (or commit) message.
type Version struct{ Tag, Date, Title string }

// GitHub is a minimal REST client.
type GitHub struct {
	Client *http.Client // nil -> &http.Client{Timeout: 10 * time.Second}
	Token  string       // optional; sent as "Authorization: Bearer <token>"
	API    string       // "" -> "https://api.github.com"
}

type statusError struct {
	path   string
	status int
}

func (e statusError) Error() string { return fmt.Sprintf("github %s: status %d", e.path, e.status) }

func (g GitHub) get(path string, v any) error {
	api, c := g.API, g.Client
	if api == "" {
		api = "https://api.github.com"
	}
	if c == nil {
		c = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, api+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError{path, resp.StatusCode}
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func day(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return strings.TrimSpace(s)
}

// ToolRepos returns non-archived, non-fork repos of owners tagged with topic, by stars.
func (g GitHub) ToolRepos(owners []string, topic string) ([]Repo, error) {
	terms := []string{url.QueryEscape("topic:" + topic)}
	for _, o := range owners {
		terms = append(terms, url.QueryEscape("user:"+o))
	}
	var res struct {
		Items []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Homepage    string `json:"homepage"`
			HTMLURL     string `json:"html_url"`
			Stars       int    `json:"stargazers_count"`
			Archived    bool   `json:"archived"`
			Fork        bool   `json:"fork"`
			License     *struct {
				SPDX string `json:"spdx_id"`
			} `json:"license"`
		} `json:"items"`
	}
	if err := g.get("/search/repositories?q="+strings.Join(terms, "+")+"&sort=stars&order=desc&per_page=100", &res); err != nil {
		return nil, err
	}
	var out []Repo
	for _, it := range res.Items {
		if it.Archived || it.Fork {
			continue
		}
		r := Repo{Name: it.Name, FullName: it.FullName, Description: it.Description,
			Homepage: it.Homepage, URL: it.HTMLURL, Stars: it.Stars}
		if it.License != nil {
			r.License = it.License.SPDX
		}
		var rels []struct {
			Tag        string `json:"tag_name"`
			Published  string `json:"published_at"`
			Prerelease bool   `json:"prerelease"`
		}
		if err := g.get("/repos/"+r.FullName+"/releases?per_page=1", &rels); err != nil {
			return nil, err
		}
		if len(rels) > 0 {
			r.Version, r.Date, r.Prerelease = rels[0].Tag, day(rels[0].Published), rels[0].Prerelease
		}
		if r.License == "" || r.License == "NOASSERTION" {
			r.License = ""
			var lic struct {
				Content string `json:"content"`
			}
			var se statusError
			err := g.get("/repos/"+r.FullName+"/license", &lic)
			switch {
			case errors.As(err, &se) && se.status == http.StatusNotFound:
			case err != nil:
				return nil, err
			default:
				text, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(lic.Content, "\n", ""))
				if err != nil {
					return nil, fmt.Errorf("github %s license: %w", r.FullName, err)
				}
				if strings.Contains(string(text), "Business Source License") {
					r.License = "BSL-1.1"
				}
			}
		}
		out = append(out, r)
	}
	return out, nil
}

// Versions returns the tags of repo ("owner/name"), newest first.
func (g GitHub) Versions(repo string) ([]Version, error) {
	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := g.get("/repos/"+repo+"/tags?per_page=100", &tags); err != nil {
		return nil, err
	}
	out := make([]Version, 0, len(tags))
	for _, t := range tags {
		var ref struct {
			Object struct {
				Type string `json:"type"`
				SHA  string `json:"sha"`
			} `json:"object"`
		}
		if err := g.get("/repos/"+repo+"/git/ref/tags/"+t.Name, &ref); err != nil {
			return nil, err
		}
		v := Version{Tag: t.Name}
		if ref.Object.Type == "tag" {
			var tag struct {
				Message string `json:"message"`
				Tagger  struct {
					Date string `json:"date"`
				} `json:"tagger"`
			}
			if err := g.get("/repos/"+repo+"/git/tags/"+ref.Object.SHA, &tag); err != nil {
				return nil, err
			}
			v.Date, v.Title = day(tag.Tagger.Date), firstLine(tag.Message)
		} else {
			var c struct {
				Commit struct {
					Message   string `json:"message"`
					Committer struct {
						Date string `json:"date"`
					} `json:"committer"`
				} `json:"commit"`
			}
			if err := g.get("/repos/"+repo+"/commits/"+t.Commit.SHA, &c); err != nil {
				return nil, err
			}
			v.Date, v.Title = day(c.Commit.Committer.Date), firstLine(c.Commit.Message)
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out, nil
}
