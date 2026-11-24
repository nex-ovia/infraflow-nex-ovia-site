package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Publisher posts drafts to one platform and returns the platform's post ID.
type Publisher interface {
	Name() string
	Publish(ctx context.Context, d Drafts) (string, error)
}

func doJSON(ctx context.Context, c *http.Client, req *http.Request, out any) (*http.Response, error) {
	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return resp, fmt.Errorf("%s %s: HTTP %d: %s", req.Method, req.URL.Path, resp.StatusCode, clip(string(body), 300))
	}
	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return resp, fmt.Errorf("%s: decoding response: %w", req.URL.Path, err)
		}
	}
	return resp, nil
}

// ---------- LinkedIn (Posts API, "Share on LinkedIn" product, w_member_social) ----------

type LinkedIn struct {
	Base, Token, Author, Version string // Author: urn:li:person:<id>; Version: YYYYMM
	HTTP                         *http.Client
}

func (LinkedIn) Name() string { return "linkedin" }

func (l LinkedIn) Publish(ctx context.Context, d Drafts) (string, error) {
	body := map[string]any{
		"author":                    l.Author,
		"commentary":                d.LinkedIn,
		"visibility":                "PUBLIC",
		"distribution":              map[string]any{"feedDistribution": "MAIN_FEED", "targetEntities": []any{}, "thirdPartyDistributionChannels": []any{}},
		"content":                   map[string]any{"article": map[string]any{"source": d.URL, "title": firstLine(d.LinkedIn)}},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, l.Base+"/rest/posts", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+l.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LinkedIn-Version", l.Version)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	resp, err := doJSON(ctx, l.HTTP, req, nil)
	if err != nil {
		return "", err
	}
	return resp.Header.Get("x-restli-id"), nil
}

func firstLine(s string) string { return strings.SplitN(s, "\n", 2)[0] }

// ---------- X (API v2, OAuth 1.0a user context; tokens don't expire) ----------

type X struct {
	Base                                         string
	ConsumerKey, ConsumerSecret, Token, TokenSec string
	HTTP                                         *http.Client
	Now                                          func() time.Time
	Nonce                                        func() string
}

func (X) Name() string { return "x" }

func (x X) Publish(ctx context.Context, d Drafts) (string, error) {
	var firstID, prev string
	for i, text := range d.X {
		body := map[string]any{"text": text}
		if prev != "" {
			body["reply"] = map[string]string{"in_reply_to_tweet_id": prev}
		}
		b, _ := json.Marshal(body)
		req, err := http.NewRequest(http.MethodPost, x.Base+"/2/tweets", bytes.NewReader(b))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", x.authHeader(req.Method, req.URL))
		var out struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if _, err := doJSON(ctx, x.HTTP, req, &out); err != nil {
			return firstID, fmt.Errorf("post %d/%d: %w", i+1, len(d.X), err)
		}
		prev = out.Data.ID
		if firstID == "" {
			firstID = prev
		}
	}
	return firstID, nil
}

// authHeader builds an OAuth 1.0a HMAC-SHA1 header. JSON bodies are not part of the signature.
func (x X) authHeader(method string, u *url.URL) string {
	now, nonce := time.Now, randomNonce
	if x.Now != nil {
		now = x.Now
	}
	if x.Nonce != nil {
		nonce = x.Nonce
	}
	p := map[string]string{
		"oauth_consumer_key":     x.ConsumerKey,
		"oauth_nonce":            nonce(),
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        strconv.FormatInt(now().Unix(), 10),
		"oauth_token":            x.Token,
		"oauth_version":          "1.0",
	}
	all := map[string]string{}
	for k, v := range p {
		all[k] = v
	}
	for k, v := range u.Query() {
		all[k] = v[0]
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, pct(k)+"="+pct(all[k]))
	}
	baseURL := u.Scheme + "://" + u.Host + u.Path
	sigBase := method + "&" + pct(baseURL) + "&" + pct(strings.Join(pairs, "&"))
	mac := hmac.New(sha1.New, []byte(pct(x.ConsumerSecret)+"&"+pct(x.TokenSec)))
	mac.Write([]byte(sigBase))
	p["oauth_signature"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	hk := make([]string, 0, len(p))
	for k := range p {
		hk = append(hk, k)
	}
	sort.Strings(hk)
	var parts []string
	for _, k := range hk {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, pct(k), pct(p[k])))
	}
	return "OAuth " + strings.Join(parts, ", ")
}

// pct is RFC 3986 percent-encoding as OAuth 1.0a requires.
func pct(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func randomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- Instagram (Graph API content publishing; Business/Creator account) ----------

type Instagram struct {
	Base, Version, UserID, Token string
	HTTP                         *http.Client
	Sleep                        func(time.Duration)
	Polls                        int
}

func (Instagram) Name() string { return "instagram" }

func (ig Instagram) Publish(ctx context.Context, d Drafts) (string, error) {
	sleep, polls := time.Sleep, 10
	if ig.Sleep != nil {
		sleep = ig.Sleep
	}
	if ig.Polls > 0 {
		polls = ig.Polls
	}
	root := ig.Base + "/" + ig.Version
	var created struct {
		ID string `json:"id"`
	}
	if err := ig.form(ctx, root+"/"+ig.UserID+"/media", url.Values{"image_url": {d.IGImage}, "caption": {d.Instagram}}, &created); err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	for i := 0; ; i++ {
		var st struct {
			StatusCode string `json:"status_code"`
		}
		req, _ := http.NewRequest(http.MethodGet, root+"/"+created.ID+"?fields=status_code", nil)
		req.Header.Set("Authorization", "Bearer "+ig.Token)
		if _, err := doJSON(ctx, ig.HTTP, req, &st); err != nil {
			return "", fmt.Errorf("container status: %w", err)
		}
		if st.StatusCode == "FINISHED" {
			break
		}
		if st.StatusCode == "ERROR" || st.StatusCode == "EXPIRED" || i+1 >= polls {
			return "", fmt.Errorf("container %s not ready: %q", created.ID, st.StatusCode)
		}
		sleep(3 * time.Second)
	}
	var pub struct {
		ID string `json:"id"`
	}
	if err := ig.form(ctx, root+"/"+ig.UserID+"/media_publish", url.Values{"creation_id": {created.ID}}, &pub); err != nil {
		return "", fmt.Errorf("publish: %w", err)
	}
	return pub.ID, nil
}

func (ig Instagram) form(ctx context.Context, endpoint string, v url.Values, out any) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+ig.Token)
	_, err = doJSON(ctx, ig.HTTP, req, out)
	return err
}
