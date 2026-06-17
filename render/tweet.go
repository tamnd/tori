// Package render turns stored records into the two human views — the kage-shape
// HTML site (render/html) and the yomi-shape Markdown archive (render/md). Both
// derive from the same view model built here (TP3), so they always agree on what
// a tweet says: the linkified text, the media list with relative local paths,
// the quoted card, the poll. The view model is pure — records and a path context
// in, a view struct out — so the renderers carry golden tests with no network
// and no clock (spec §14).
package render

import (
	"fmt"
	"html/template"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/tori/repo"
	"github.com/tamnd/x-cli/x"
)

// Context carries what the view model needs to resolve references: the set of
// tweet ids that are in the archive (so an in-archive link is relative and an
// outside link stays absolute), and a lookup from a media source URL to its
// localised repo-relative path (empty when the item is not on disk).
type Context struct {
	InArchive map[string]bool   // tweet id -> present
	MediaPath map[string]string // source URL -> repo-relative local path
	// FromPage is the repo-relative path of the page being rendered, so media
	// and cross-tweet links can be rewritten relative to it (spec §6.4).
	FromPage string
}

// NewContext builds a render context from a record set and a media result. The
// caller sets FromPage per page before rendering.
func NewContext(tweets []*x.Tweet, assets []repo.Asset) *Context {
	in := make(map[string]bool, len(tweets))
	for _, t := range tweets {
		in[t.ID] = true
	}
	mp := make(map[string]string, len(assets))
	for _, a := range assets {
		if a.Status == "local" && a.Path != "" {
			mp[a.Source] = a.Path
		}
	}
	return &Context{InArchive: in, MediaPath: mp}
}

// MediaView is one renderable media item with its resolved local (or remote)
// source and a flag for whether it is on disk.
type MediaView struct {
	Type     string // photo|video|gif
	Src      string // relative local path when localised, else the remote URL
	Local    bool
	AltText  string
	Width    int
	Height   int
	Poster   string // relative/remote preview image for video
	Unavail  bool   // true when the media could not be localised
	Duration int
}

// PollOptionView is one poll choice with a percentage for the result bar.
type PollOptionView struct {
	Label   string
	Votes   int
	Percent int
}

// TweetView is the shared, presentation-agnostic view of one tweet.
type TweetView struct {
	ID          string
	URL         string // canonical x.com permalink (the source)
	Permalink   string // page-relative link to this tweet's own local page
	AuthorName  string
	Handle      string
	Verified    bool
	AvatarSrc   string
	AvatarLocal bool
	CreatedAt   time.Time
	Stamp       string        // formatted timestamp
	HTMLBody    template.HTML // text with entities linkified, HTML-escaped (for the html renderer)
	TextBody    string        // raw text (for the markdown renderer)
	Media       []MediaView
	Poll        []PollOptionView
	PollStatus  string
	Metrics     x.Metrics
	IsReply     bool
	ReplyToUser string
	ReplyToID   string
	ReplyToRel  string // relative link to the parent when in-archive, else absolute, else ""
	IsRetweet   bool
	IsQuote     bool
	Quoted      *TweetView
	Lang        string
	Source      string
}

// Build constructs a TweetView for one tweet under the given context. The body
// is linkified two ways: HTMLBody for the HTML renderer (escaped, anchor tags)
// and TextBody for Markdown (left raw; the md renderer links entities itself).
func (c *Context) Build(t *x.Tweet) TweetView {
	if t == nil {
		return TweetView{}
	}
	tv := TweetView{
		ID:          t.ID,
		URL:         t.URL,
		CreatedAt:   t.CreatedAt,
		Stamp:       t.CreatedAt.UTC().Format("2006-01-02 15:04 MST"),
		TextBody:    t.Text,
		HTMLBody:    linkifyHTML(t.Text, t.Entities, c),
		Metrics:     t.Metrics,
		IsReply:     t.IsReply,
		ReplyToUser: t.ReplyToUser,
		ReplyToID:   t.ReplyTo,
		IsRetweet:   t.IsRetweet,
		IsQuote:     t.IsQuote,
		Lang:        t.Lang,
		Source:      t.Source,
	}
	if t.Author != nil {
		tv.AuthorName = t.Author.Name
		tv.Handle = t.Author.Username
		tv.Verified = t.Author.Verified
		tv.AvatarSrc, tv.AvatarLocal = c.resolveURL(t.Author.ProfileImage)
	}
	if tv.AuthorName == "" {
		tv.AuthorName = tv.Handle
	}
	if t.ReplyTo != "" {
		tv.ReplyToRel = c.linkToTweet(t.ReplyTo)
	}
	// A page-relative link to this tweet's own local page, so the index and
	// thread views are navigable offline. Empty on the tweet's own page (a
	// self-link) and for a quoted/retweeted tweet that is not in the archive.
	if c.InArchive[t.ID] {
		if rel := c.linkToTweet(t.ID); rel != "" && rel != path.Base(c.FromPage) {
			tv.Permalink = rel
		}
	}
	tv.Media = c.mediaViews(t)
	if t.Poll != nil {
		tv.Poll, tv.PollStatus = pollViews(t.Poll)
	}
	if t.Quoted != nil {
		q := c.Build(t.Quoted)
		tv.Quoted = &q
	}
	if t.Retweeted != nil && t.Quoted == nil {
		// A pure retweet renders the original as the quoted card so the reader
		// sees what was retweeted.
		q := c.Build(t.Retweeted)
		tv.Quoted = &q
	}
	return tv
}

// mediaViews resolves each media item to a local-or-remote src relative to the
// page being rendered.
func (c *Context) mediaViews(t *x.Tweet) []MediaView {
	var out []MediaView
	for _, m := range t.Media {
		mv := MediaView{AltText: m.AltText, Width: m.Width, Height: m.Height, Duration: m.Duration}
		switch m.Type {
		case "photo":
			mv.Type = "photo"
			mv.Src, mv.Local = c.resolveURL(m.URL)
		case "animated_gif", "gif":
			mv.Type = "gif"
			mv.Src, mv.Local = c.resolveVariant(m)
			mv.Poster, _ = c.resolveURL(m.Preview)
		case "video":
			mv.Type = "video"
			mv.Src, mv.Local = c.resolveVariant(m)
			mv.Poster, _ = c.resolveURL(m.Preview)
		default:
			continue
		}
		if mv.Src == "" {
			mv.Unavail = true
		}
		out = append(out, mv)
	}
	return out
}

// resolveVariant resolves a video/gif to its localised file when present, else
// to a best remote rendition so the page still links to something playable.
func (c *Context) resolveVariant(m x.Media) (string, bool) {
	if url, _ := pickBestVariant(m); url != "" {
		if rel, ok := c.localFor(url); ok {
			return rel, true
		}
		return url, false
	}
	if rel, ok := c.localFor(m.URL); ok {
		return rel, true
	}
	return m.URL, false
}

// resolveURL maps a source URL to a page-relative local path when it is on disk,
// else returns the URL unchanged (an outside reference stays absolute).
func (c *Context) resolveURL(srcURL string) (string, bool) {
	if srcURL == "" {
		return "", false
	}
	if rel, ok := c.localFor(srcURL); ok {
		return rel, true
	}
	return srcURL, false
}

// localFor returns the page-relative path to a localised media source, if held.
func (c *Context) localFor(srcURL string) (string, bool) {
	repoPath, ok := c.MediaPath[srcURL]
	if !ok || repoPath == "" {
		return "", false
	}
	if c.FromPage == "" {
		return repoPath, true
	}
	return repo.Rel(c.FromPage, repoPath), true
}

// MediaSrc resolves a source URL to a path relative to page when the asset is
// localised, else reports false so the caller can fall back to the remote URL.
// It is the exported entry the renderers use for profile avatars and banners,
// which are not part of a tweet's media list.
func (c *Context) MediaSrc(srcURL, page string) (string, bool) {
	repoPath, ok := c.MediaPath[srcURL]
	if !ok || repoPath == "" {
		return "", false
	}
	if page == "" {
		return repoPath, true
	}
	return repo.Rel(page, repoPath), true
}

// linkToTweet returns a page-relative link to another tweet's HTML page when it
// is in the archive, else the absolute x.com permalink, else "".
func (c *Context) linkToTweet(id string) string {
	if id == "" {
		return ""
	}
	if c.InArchive[id] {
		target := repo.TweetHTML(id)
		if c.FromPage == "" {
			return target
		}
		return repo.Rel(c.FromPage, target)
	}
	return x.TweetURL("", id)
}

// pollViews converts a poll to result bars with integer percentages summing to
// roughly 100.
func pollViews(p *x.Poll) ([]PollOptionView, string) {
	total := 0
	for _, o := range p.Options {
		total += o.Votes
	}
	opts := make([]PollOptionView, 0, len(p.Options))
	ordered := append([]x.PollOption(nil), p.Options...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Position < ordered[j].Position })
	for _, o := range ordered {
		pct := 0
		if total > 0 {
			pct = int(float64(o.Votes)*100.0/float64(total) + 0.5)
		}
		opts = append(opts, PollOptionView{Label: o.Label, Votes: o.Votes, Percent: pct})
	}
	return opts, p.VotingStatus
}

// pickBestVariant returns the highest-bitrate mp4 rendition URL of a video/gif.
func pickBestVariant(m x.Media) (string, int) {
	best := ""
	bestBr := -1
	for _, v := range m.Variants {
		if !strings.Contains(v.ContentType, "mp4") {
			continue
		}
		if v.Bitrate > bestBr {
			bestBr = v.Bitrate
			best = v.URL
		}
	}
	return best, bestBr
}

// FormatCount renders an engagement count compactly (1.2K, 3.4M), the way X
// shows it, for the metric row.
func FormatCount(n int) string {
	switch {
	case n >= 1_000_000:
		return trimZero(float64(n)/1_000_000) + "M"
	case n >= 1_000:
		return trimZero(float64(n)/1_000) + "K"
	default:
		return fmt.Sprintf("%d", n)
	}
}

func trimZero(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	s = strings.TrimSuffix(s, ".0")
	return s
}
