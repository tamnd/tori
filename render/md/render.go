// Package md renders the yomi-shape Markdown archive: a plain-text mirror of the
// repository that reads naturally, greps, and diffs (spec §10). The repository
// home is README.md; each tweet is md/<id>.md and each reconstructed thread is
// threads/<root>.md. Markdown derives from the same stored records as the HTML
// site, so the two views always agree (TP3). Output is deterministic — no clock,
// no map iteration in the text — so golden tests run with no network.
package md

import (
	"fmt"
	"strings"

	"github.com/tamnd/tori/render"
	"github.com/tamnd/tori/repo"
	"github.com/tamnd/tori/thread"
	"github.com/tamnd/x-cli/x"
)

// Renderer builds the Markdown views over one repository's records and media.
type Renderer struct {
	ctx     *render.Context
	profile *x.User
	footer  string
	title   string
}

// New builds a Markdown renderer. footer carries the capture stamp; title is the
// repository display name used in the README heading.
func New(tweets []*x.Tweet, assets []repo.Asset, profile *x.User, footer, title string) *Renderer {
	return &Renderer{
		ctx:     render.NewContext(tweets, assets),
		profile: profile,
		footer:  footer,
		title:   title,
	}
}

// Tweet renders one tweet as a standalone Markdown document for md/<id>.md.
func (r *Renderer) Tweet(t *x.Tweet) string {
	page := repo.TweetMD(t.ID)
	r.ctx.FromPage = page
	var b strings.Builder
	r.writeTweet(&b, r.ctx.Build(t), 0)
	r.writeFooter(&b, repo.Rel(page, repo.ReadmeFile))
	return b.String()
}

// Thread renders a reconstructed conversation for threads/<root>.md.
func (r *Renderer) Thread(th thread.Thread) string {
	page := repo.ThreadMD(th.RootID)
	r.ctx.FromPage = page
	var b strings.Builder
	fmt.Fprintf(&b, "# Thread by @%s\n\n", rootHandle(th))
	for i, t := range th.Tweets {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		r.writeTweet(&b, r.ctx.Build(t), 0)
	}
	r.writeFooter(&b, repo.Rel(page, repo.ReadmeFile))
	return b.String()
}

// Index renders README.md: the profile header (when captured) followed by a
// reverse-chronological list of captured threads, each linking to its Markdown
// page. heading/subheading label a non-profile capture.
func (r *Renderer) Index(threads []thread.Thread, heading, subheading string) string {
	page := repo.ReadmeFile
	r.ctx.FromPage = page
	var b strings.Builder
	if r.profile != nil {
		r.writeProfile(&b)
	} else {
		fmt.Fprintf(&b, "# %s\n\n", mdEscape(r.title))
	}
	if heading != "" {
		fmt.Fprintf(&b, "## %s\n\n", mdEscape(heading))
		if subheading != "" {
			fmt.Fprintf(&b, "%s\n\n", mdEscape(subheading))
		}
	}
	fmt.Fprintf(&b, "%d posts archived.\n\n", len(threads))
	for _, th := range threads {
		v := r.ctx.Build(th.Root)
		link := repo.Rel(page, repo.TweetMD(th.RootID))
		if !th.Standalone() {
			link = repo.Rel(page, repo.ThreadMD(th.RootID))
		}
		fmt.Fprintf(&b, "- [%s](%s) — %s\n", v.Stamp, link, summary(v.TextBody))
	}
	b.WriteString("\n")
	b.WriteString(r.footer)
	b.WriteString("\n")
	return b.String()
}

func (r *Renderer) writeProfile(b *strings.Builder) {
	u := r.profile
	name := u.Name
	if name == "" {
		name = u.Username
	}
	fmt.Fprintf(b, "# %s (@%s)\n\n", mdEscape(name), u.Username)
	if u.Description != "" {
		fmt.Fprintf(b, "%s\n\n", linkifyMD(u.Description))
	}
	fmt.Fprintf(b, "**%s** Following · **%s** Followers", render.FormatCount(u.Metrics.Following), render.FormatCount(u.Metrics.Followers))
	if u.Metrics.Tweets > 0 {
		fmt.Fprintf(b, " · **%s** Posts", render.FormatCount(u.Metrics.Tweets))
	}
	b.WriteString("\n\n")
}

// writeTweet writes one tweet view. indent is reserved for nested quote bodies,
// which are rendered as blockquotes instead.
func (r *Renderer) writeTweet(b *strings.Builder, v render.TweetView, _ int) {
	if v.IsRetweet {
		fmt.Fprintf(b, "> reposted by @%s\n\n", v.Handle)
	}
	if v.IsReply && v.ReplyToUser != "" {
		fmt.Fprintf(b, "*Replying to @%s*\n\n", v.ReplyToUser)
	}
	fmt.Fprintf(b, "**%s** @%s · [%s](%s)\n\n", mdEscape(v.AuthorName), v.Handle, v.Stamp, v.URL)
	if body := strings.TrimSpace(linkifyMD(v.TextBody)); body != "" {
		b.WriteString(body)
		b.WriteString("\n\n")
	}
	r.writeMedia(b, v.Media)
	r.writePoll(b, v.Poll, v.PollStatus)
	if v.Quoted != nil {
		r.writeQuote(b, v.Quoted)
	}
	fmt.Fprintf(b, "💬 %s · 🔁 %s · ♥ %s",
		render.FormatCount(v.Metrics.Replies),
		render.FormatCount(v.Metrics.Retweets),
		render.FormatCount(v.Metrics.Likes))
	if v.Metrics.Impressions > 0 {
		fmt.Fprintf(b, " · 👁 %s", render.FormatCount(v.Metrics.Impressions))
	}
	b.WriteString("\n")
}

func (r *Renderer) writeMedia(b *strings.Builder, media []render.MediaView) {
	for _, m := range media {
		switch {
		case m.Unavail:
			b.WriteString("*(media unavailable)*\n\n")
		case m.Type == "photo":
			alt := m.AltText
			if alt == "" {
				alt = "photo"
			}
			fmt.Fprintf(b, "![%s](%s)\n\n", mdEscape(alt), m.Src)
		default:
			label := m.Type
			if label == "" {
				label = "video"
			}
			fmt.Fprintf(b, "[▶ %s](%s)\n\n", label, m.Src)
		}
	}
}

func (r *Renderer) writePoll(b *strings.Builder, opts []render.PollOptionView, status string) {
	if len(opts) == 0 {
		return
	}
	for _, o := range opts {
		fmt.Fprintf(b, "- %s — %d%%\n", mdEscape(o.Label), o.Percent)
	}
	if status != "" {
		fmt.Fprintf(b, "\n*%s*\n", status)
	}
	b.WriteString("\n")
}

func (r *Renderer) writeQuote(b *strings.Builder, q *render.TweetView) {
	fmt.Fprintf(b, "> **%s** @%s\n", mdEscape(q.AuthorName), q.Handle)
	for _, line := range strings.Split(strings.TrimRight(linkifyMD(q.TextBody), "\n"), "\n") {
		fmt.Fprintf(b, "> %s\n", line)
	}
	b.WriteString("\n")
}

func (r *Renderer) writeFooter(b *strings.Builder, homeRel string) {
	fmt.Fprintf(b, "\n---\n\n[← archive home](%s)\n\n%s\n", homeRel, r.footer)
}

// summary trims a tweet body to a single short line for the index list.
func summary(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 100 {
		text = text[:100] + "…"
	}
	if text == "" {
		return "(no text)"
	}
	return mdEscape(text)
}

// linkifyMD is the package-local Markdown linkifier, kept here so the md renderer
// owns its surface treatment.
func linkifyMD(text string) string { return render.LinkifyMarkdown(text) }

// mdEscape neutralises the few characters that would break inline Markdown when
// they appear in a name, label, or summary. Tweet bodies are left to linkifyMD,
// which keeps them readable.
func mdEscape(s string) string {
	r := strings.NewReplacer(
		"[", "\\[", "]", "\\]",
		"*", "\\*", "_", "\\_",
		"`", "\\`", "|", "\\|",
		"<", "&lt;", ">", "&gt;",
	)
	return r.Replace(s)
}

func rootHandle(th thread.Thread) string {
	if th.Root != nil && th.Root.Author != nil {
		return th.Root.Author.Username
	}
	return ""
}
