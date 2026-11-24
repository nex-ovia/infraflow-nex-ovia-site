package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// RepoStats are the live figures for one GitHub repo.
type RepoStats struct {
	Releases  int
	Downloads int
	Latest    string // "<tag> · <YYYY-MM-DD>" of the newest release
}

var metricRe = regexp.MustCompile(`(<[a-zA-Z0-9]+[^>]*\sdata-metric="([^"]+)"[^>]*>)([^<]*)(</)`)
var metricDateRe = regexp.MustCompile(`(<[a-zA-Z0-9]+[^>]*\sdata-metric-date[^>]*>)([^<]*)(</)`)

// FillMetrics replaces the text of each data-metric="<repo>:<field>" element with the fetched value.
// On a fetch error or unknown field the existing text stays (static fallback).
func FillMetrics(html string, fetch func(repo string) (RepoStats, error)) string {
	return metricRe.ReplaceAllStringFunc(html, func(m string) string {
		g := metricRe.FindStringSubmatch(m)
		repo, field, ok := strings.Cut(g[2], ":")
		if !ok {
			return m
		}
		st, err := fetch(repo)
		if err != nil {
			return m
		}
		var v string
		switch field {
		case "releases":
			v = fmt.Sprint(st.Releases)
		case "downloads":
			v = fmt.Sprint(st.Downloads)
		case "latest":
			v = st.Latest
		default:
			return m
		}
		return g[1] + v + g[4]
	})
}

// FillMetricDate sets the text of the data-metric-date element to date.
func FillMetricDate(html, date string) string {
	return metricDateRe.ReplaceAllString(html, "${1}"+date+"${3}")
}

// GitHubStats returns a fetcher that reads release stats from the GitHub API, once per repo per build.
func GitHubStats(client *http.Client, token string) func(string) (RepoStats, error) {
	return githubStats(client, token, "https://api.github.com")
}

func githubStats(client *http.Client, token, api string) func(string) (RepoStats, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	type res struct {
		st  RepoStats
		err error
	}
	cache := map[string]res{}
	get := func(repo string) (RepoStats, error) {
		req, err := http.NewRequest("GET", api+"/repos/"+repo+"/releases?per_page=100", nil)
		if err != nil {
			return RepoStats{}, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return RepoStats{}, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return RepoStats{}, fmt.Errorf("github %s: %s", repo, resp.Status)
		}
		var rel []struct {
			Tag         string `json:"tag_name"`
			PublishedAt string `json:"published_at"`
			Assets      []struct {
				Count int `json:"download_count"`
			} `json:"assets"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			return RepoStats{}, err
		}
		st := RepoStats{Releases: len(rel)}
		for _, r := range rel {
			for _, a := range r.Assets {
				st.Downloads += a.Count
			}
		}
		if len(rel) > 0 {
			d := rel[0].PublishedAt
			if len(d) >= 10 {
				d = d[:10]
			}
			st.Latest = rel[0].Tag + " · " + d
		}
		return st, nil
	}
	return func(repo string) (RepoStats, error) {
		if c, ok := cache[repo]; ok {
			return c.st, c.err
		}
		st, err := get(repo)
		cache[repo] = res{st, err}
		return st, err
	}
}
