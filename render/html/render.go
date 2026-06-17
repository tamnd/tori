// Package html renders the kage-shape static site from stored records: an
// inert, self-contained set of pages that look like X and open with the network
// unplugged (spec §9). No <script>, no on* handlers, no remote fonts — a
// photograph, not a program. Templates are embedded so the binary needs no asset
// directory at runtime, and html/template auto-escapes every value.
package html

import (
	"bytes"
	"embed"
	"html/template"

	"github.com/tamnd/tori/render"
	"github.com/tamnd/tori/repo"
	"github.com/tamnd/tori/thread"
	"github.com/tamnd/x-cli/x"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

//go:embed assets/tori.css
var cssBytes []byte

// CSS returns the embedded stylesheet bytes so the caller can write it into the
// repository's _assets directory.
func CSS() []byte { return cssBytes }

var tmpl = template.Must(template.New("tori").Funcs(template.FuncMap{
	"count": render.FormatCount,
}).ParseFS(templatesFS, "templates/*.tmpl"))

// Renderer holds the shared context for one repository's pages. It is
// single-use per render pass and not safe for concurrent pages (it sets the
// per-page FromPage on the shared context before each build).
type Renderer struct {
	ctx     *render.Context
	profile *x.User
	tweets  []*x.Tweet
	footer  string
	navTit  string
}

// New builds a renderer over a record set, the localised media, and the profile.
// footer is the page footer line (carrying the --date capture stamp); navTitle
// is the repository's display name shown in the top nav.
func New(tweets []*x.Tweet, assets []repo.Asset, profile *x.User, footer, navTitle string) *Renderer {
	return &Renderer{
		ctx:     render.NewContext(tweets, assets),
		profile: profile,
		tweets:  tweets,
		footer:  footer,
		navTit:  navTitle,
	}
}

// card pairs a tweet view with a root flag for the thread template.
type card struct {
	View render.TweetView
	Root bool
}

type profileView struct {
	Name      string
	Handle    string
	Verified  bool
	Bio       template.HTML
	AvatarSrc string
	BannerSrc string
	Following int
	Followers int
	Tweets    int
}

type pageData struct {
	Title      string
	CSSHref    string
	HomeHref   string
	NavTitle   string
	Footer     string
	Profile    *profileView
	Heading    string
	SubHeading string
	Single     bool
	IsThread   bool
	Cards      []card
	SingleCard card
}

// TweetPage renders one tweet as a standalone page at html/<id>.html.
func (r *Renderer) TweetPage(t *x.Tweet) (string, error) {
	page := repo.TweetHTML(t.ID)
	r.ctx.FromPage = page
	v := r.ctx.Build(t)
	data := pageData{
		Title:      pageTitle(v),
		CSSHref:    repo.Rel(page, repo.CSSFile),
		HomeHref:   repo.Rel(page, repo.IndexFile),
		NavTitle:   r.navTit,
		Footer:     r.footer,
		Single:     true,
		SingleCard: card{View: v, Root: true},
	}
	return r.exec(data)
}

// ThreadPage renders a reconstructed conversation at threads/<root>.html.
func (r *Renderer) ThreadPage(th thread.Thread) (string, error) {
	page := repo.ThreadHTML(th.RootID)
	r.ctx.FromPage = page
	cards := make([]card, 0, len(th.Tweets))
	for _, t := range th.Tweets {
		cards = append(cards, card{View: r.ctx.Build(t), Root: t.ID == th.RootID})
	}
	data := pageData{
		Title:    "Thread by @" + rootHandle(th),
		CSSHref:  repo.Rel(page, repo.CSSFile),
		HomeHref: repo.Rel(page, repo.IndexFile),
		NavTitle: r.navTit,
		Footer:   r.footer,
		Single:   true,
		IsThread: true,
		Cards:    cards,
	}
	return r.exec(data)
}

// Index renders the repository home at index.html: the profile header (when a
// profile was captured) followed by a reverse-chronological list of the captured
// threads and standalone tweets, each linking to its page. heading/subheading
// label a non-profile capture (a search or a list).
func (r *Renderer) Index(threads []thread.Thread, heading, subheading string) (string, error) {
	page := repo.IndexFile
	r.ctx.FromPage = page
	cards := make([]card, 0, len(threads))
	for _, th := range threads {
		// The index shows each thread's root; the thread page holds the replies.
		cards = append(cards, card{View: r.ctx.Build(th.Root)})
	}
	data := pageData{
		Title:      r.navTit,
		CSSHref:    repo.Rel(page, repo.CSSFile),
		HomeHref:   ".",
		NavTitle:   r.navTit,
		Footer:     r.footer,
		Heading:    heading,
		SubHeading: subheading,
		Cards:      cards,
	}
	if r.profile != nil {
		data.Profile = r.profileView(page)
	}
	return r.exec(data)
}

func (r *Renderer) profileView(page string) *profileView {
	u := r.profile
	pv := &profileView{
		Name:      u.Name,
		Handle:    u.Username,
		Verified:  u.Verified,
		Bio:       render.BioHTML(u.Description),
		Following: u.Metrics.Following,
		Followers: u.Metrics.Followers,
		Tweets:    u.Metrics.Tweets,
	}
	if u.Name == "" {
		pv.Name = u.Username
	}
	if src, ok := r.ctx.MediaSrc(u.ProfileImage, page); ok {
		pv.AvatarSrc = src
	} else if u.ProfileImage != "" {
		pv.AvatarSrc = u.ProfileImage
	}
	if src, ok := r.ctx.MediaSrc(u.ProfileBanner, page); ok {
		pv.BannerSrc = src
	} else if u.ProfileBanner != "" {
		pv.BannerSrc = u.ProfileBanner
	}
	return pv
}

func (r *Renderer) exec(data pageData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func pageTitle(v render.TweetView) string {
	body := v.TextBody
	if len(body) > 70 {
		body = body[:70] + "…"
	}
	if body == "" {
		return "Post by @" + v.Handle
	}
	return body + " — @" + v.Handle
}

func rootHandle(th thread.Thread) string {
	if th.Root != nil && th.Root.Author != nil {
		return th.Root.Author.Username
	}
	return ""
}
