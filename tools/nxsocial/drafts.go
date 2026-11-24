package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/nex-ovia/infraflow-nex-ovia-site/tools/content"
)

// Platform limits (characters).
const (
	linkedInMax  = 3000
	xMax         = 280
	xURLWeight   = 23 // X counts every link as 23 characters (t.co)
	instagramMax = 2200
	igMaxTags    = 30
)

// Drafts is the ready-to-post copy for one blog post.
type Drafts struct {
	Slug      string
	URL       string   // canonical post URL
	LinkedIn  string   // single post with link
	X         []string // thread; the link is in the first post
	Instagram string   // caption; links aren't clickable on Instagram, so it points to the bio
	IGImage   string   // public URL of the 1080x1350 card
	OGImage   string   // public URL of the 1200x630 card
}

var nonTagRe = regexp.MustCompile(`[^A-Za-z0-9]`)

// hashtags turns post tags into #Tags (letters and digits only).
func hashtags(tags []string, max int) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range tags {
		h := nonTagRe.ReplaceAllString(t, "")
		if h == "" || seen[strings.ToLower(h)] {
			continue
		}
		seen[strings.ToLower(h)] = true
		out = append(out, "#"+h)
		if len(out) == max {
			break
		}
	}
	return out
}

// MakeDrafts builds per-platform copy for a post published at base (e.g. https://www.nex-ovia.com).
func MakeDrafts(p content.Post, base string) Drafts {
	base = strings.TrimRight(base, "/")
	d := Drafts{Slug: p.Slug, URL: base + "/blog/" + p.Slug + "/",
		IGImage: base + "/og/" + p.Slug + "-ig.png", OGImage: base + "/og/" + p.Slug + ".png"}
	teaser := content.Teaser(p)
	tags := hashtags(p.Tags, igMaxTags)

	d.LinkedIn = clip(strings.Join(nonEmpty(p.Title, teaser, "Read it here: "+d.URL, strings.Join(first(tags, 5), " ")), "\n\n"), linkedInMax)

	head := p.Title + "\n\n" + d.URL
	d.X = []string{clipX(head)}
	if teaser != "" && teaser != p.Title {
		d.X = append(d.X, splitX(teaser)...)
	}

	d.Instagram = clip(strings.Join(nonEmpty(p.Title, teaser, "Full post: link in bio → nex-ovia.com/blog", strings.Join(tags, " ")), "\n\n"), instagramMax)
	return d
}

// XLen counts a post the way X does: every http(s) URL weighs 23 characters.
func XLen(s string) int {
	n := 0
	for _, w := range strings.SplitAfter(s, " ") {
		for _, part := range strings.SplitAfter(w, "\n") {
			trim := strings.TrimRight(part, " \n")
			if strings.HasPrefix(trim, "http://") || strings.HasPrefix(trim, "https://") {
				n += xURLWeight + utf8.RuneCountInString(part) - utf8.RuneCountInString(trim)
				continue
			}
			n += utf8.RuneCountInString(part)
		}
	}
	return n
}

// clipX shortens the title part of "title\n\nurl" until it fits.
func clipX(s string) string {
	for XLen(s) > xMax {
		i := strings.Index(s, "\n\n")
		if i <= 1 {
			return clip(s, xMax)
		}
		title := []rune(s[:i])
		s = strings.TrimRight(string(title[:len(title)-2]), " ") + "…" + s[i:]
	}
	return s
}

// splitX splits text into posts of at most 280 characters on word boundaries.
func splitX(text string) []string {
	var out []string
	cur := ""
	for _, w := range strings.Fields(text) {
		try := strings.TrimSpace(cur + " " + w)
		if XLen(try) <= xMax {
			cur = try
			continue
		}
		if cur != "" {
			out = append(out, cur)
		}
		cur = clip(w, xMax)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func clip(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimRight(string(r[:max-1]), " \n") + "…"
}

func nonEmpty(xs ...string) []string {
	var out []string
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			out = append(out, x)
		}
	}
	return out
}

func first(xs []string, n int) []string {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}

// Render writes all drafts as one human-readable text block (the CI artifact).
func (d Drafts) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n%s\n\n## LinkedIn (%d/%d)\n%s\n\n## X thread\n", d.Slug, d.URL, utf8.RuneCountInString(d.LinkedIn), linkedInMax, d.LinkedIn)
	for i, t := range d.X {
		fmt.Fprintf(&b, "[%d/%d · %d chars]\n%s\n\n", i+1, len(d.X), XLen(t), t)
	}
	fmt.Fprintf(&b, "## Instagram (%d/%d)\nimage: %s\n%s\n", utf8.RuneCountInString(d.Instagram), instagramMax, d.IGImage, d.Instagram)
	return b.String()
}
