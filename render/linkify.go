package render

import (
	"html"
	"html/template"
	"regexp"
	"strings"

	"github.com/tamnd/x-cli/x"
)

// reEntity recognises the surface features X itself links, in one pass over the
// raw (unescaped) tweet text: a URL, a mention, a hashtag, or a cashtag. URL is
// first so an @ or # inside a URL is not re-linked. The text is matched raw, not
// HTML-escaped, so the numeric character references escaping would introduce
// (an apostrophe becomes &#39;, a quote &#34;) can never be mistaken for a
// hashtag. Each gap between matches is escaped as it is emitted, and the anchors
// this file writes are the only markup the output carries, so the page stays
// inert (no script, no event handlers) per kage's posture.
var reEntity = regexp.MustCompile(`(https?://[^\s<]+)|@(\w{1,15})|#(\w+)|\$([A-Za-z]{1,6})`)

// linkifyHTML turns a tweet's text into safe HTML: mentions, hashtags, cashtags,
// and bare URLs become anchors to x.com, everything else is HTML-escaped, and
// newlines become <br>. It returns template.HTML because the output is already
// escaped and the template must emit it verbatim rather than escape it again.
func linkifyHTML(text string, _ x.Entities, _ *Context) template.HTML {
	var b strings.Builder
	last := 0
	for _, m := range reEntity.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[0], m[1]
		isURL := m[2] >= 0
		// A mention/hashtag/cashtag only counts at a boundary: not glued to the
		// end of a word (so foo#bar and an e-mail's @ are left as plain text).
		if !isURL && start > 0 && isWordByte(text[start-1]) {
			continue // leave it in the next gap, where it is escaped as text
		}
		b.WriteString(html.EscapeString(text[last:start]))
		switch {
		case isURL:
			writeURLAnchor(&b, text[start:end])
		case m[4] >= 0:
			h := text[m[4]:m[5]]
			b.WriteString(`<a href="https://x.com/` + h + `" rel="nofollow noopener">@` + h + `</a>`)
		case m[6] >= 0:
			tag := text[m[6]:m[7]]
			b.WriteString(`<a href="https://x.com/hashtag/` + tag + `" rel="nofollow noopener">#` + tag + `</a>`)
		case m[8] >= 0:
			sym := text[m[8]:m[9]]
			b.WriteString(`<a href="https://x.com/search?q=%24` + sym + `" rel="nofollow noopener">$` + sym + `</a>`)
		}
		last = end
	}
	b.WriteString(html.EscapeString(text[last:]))
	// Newlines are left as-is: the .text rule renders with white-space: pre-wrap,
	// so a raw newline is a line break and intentional runs of spaces (ASCII art,
	// code) survive. Adding <br> on top would double every break.
	return template.HTML(b.String())
}

// writeURLAnchor emits a URL as an anchor, keeping trailing punctuation outside
// the link, and escaping the URL for both the href and the visible text.
func writeURLAnchor(b *strings.Builder, u string) {
	trail := ""
	for len(u) > 0 {
		last := u[len(u)-1]
		if strings.IndexByte(").,!?;:", last) >= 0 {
			trail = string(last) + trail
			u = u[:len(u)-1]
			continue
		}
		break
	}
	esc := html.EscapeString(u)
	b.WriteString(`<a href="` + esc + `" rel="nofollow noopener">` + esc + `</a>`)
	b.WriteString(html.EscapeString(trail))
}

// isWordByte reports whether b is an ASCII word character (letter, digit, or _).
func isWordByte(b byte) bool {
	return b == '_' ||
		(b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

// BioHTML linkifies a profile description the same inert way as a tweet body.
// It is exported because the HTML renderer builds the profile header outside the
// tweet view model.
func BioHTML(text string) template.HTML { return linkifyHTML(text, x.Entities{}, nil) }

// LinkifyMarkdown turns the same surface features into Markdown links for the
// Markdown renderer. URLs become autolinks; mentions/hashtags/cashtags link to
// x.com. The text is left otherwise verbatim so it reads naturally and greps.
// It runs over raw text and uses the same boundary rule as the HTML path.
func LinkifyMarkdown(text string) string {
	var b strings.Builder
	last := 0
	for _, m := range reEntity.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[0], m[1]
		isURL := m[2] >= 0
		if !isURL && start > 0 && isWordByte(text[start-1]) {
			continue
		}
		b.WriteString(text[last:start])
		switch {
		case isURL:
			u, trail := splitTrail(text[start:end])
			b.WriteString("[" + u + "](" + u + ")" + trail)
		case m[4] >= 0:
			h := text[m[4]:m[5]]
			b.WriteString("[@" + h + "](https://x.com/" + h + ")")
		case m[6] >= 0:
			tag := text[m[6]:m[7]]
			b.WriteString("[#" + tag + "](https://x.com/hashtag/" + tag + ")")
		case m[8] >= 0:
			sym := text[m[8]:m[9]]
			b.WriteString("[$" + sym + "](https://x.com/search?q=%24" + sym + ")")
		}
		last = end
	}
	b.WriteString(text[last:])
	return b.String()
}

// splitTrail peels trailing sentence punctuation off a URL so it stays outside
// the link.
func splitTrail(u string) (string, string) {
	trail := ""
	for len(u) > 0 {
		last := u[len(u)-1]
		if strings.IndexByte(").,!?;:", last) >= 0 {
			trail = string(last) + trail
			u = u[:len(u)-1]
			continue
		}
		break
	}
	return u, trail
}
