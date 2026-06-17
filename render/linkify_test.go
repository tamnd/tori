package render

import (
	"strings"
	"testing"

	"github.com/tamnd/x-cli/x"
)

// TestLinkifyHTMLEntitiesAreNotHashtags guards the bug where the linkifier ran
// over already-escaped text: an apostrophe became &#39; and a quote &#34;, and
// the hashtag pattern then matched the #39 / #34 inside those numeric character
// references and wrapped them in <a href="https://x.com/hashtag/39"> links. The
// fix matches over raw text instead. Plain punctuation must survive as escaped
// text with no spurious links.
func TestLinkifyHTMLEntitiesAreNotHashtags(t *testing.T) {
	got := string(linkifyHTML(`it's SOTA, "gets it" and done`, x.Entities{}, nil))
	if strings.Contains(got, "hashtag/39") || strings.Contains(got, "hashtag/34") {
		t.Fatalf("apostrophe/quote escaped to a numeric entity got linked as a hashtag: %q", got)
	}
	if !strings.Contains(got, "it&#39;s") {
		t.Fatalf("apostrophe should be HTML-escaped, got %q", got)
	}
	if !strings.Contains(got, "&#34;gets it&#34;") {
		t.Fatalf("quotes should be HTML-escaped, got %q", got)
	}
	if strings.Contains(got, "<a") {
		t.Fatalf("text with no real entities should carry no anchors, got %q", got)
	}
}

func TestLinkifyHTMLRealEntities(t *testing.T) {
	got := string(linkifyHTML("hey @jack see #golang and $TSLA https://x.com/x", x.Entities{}, nil))
	for _, want := range []string{
		`<a href="https://x.com/jack" rel="nofollow noopener">@jack</a>`,
		`<a href="https://x.com/hashtag/golang" rel="nofollow noopener">#golang</a>`,
		`<a href="https://x.com/search?q=%24TSLA" rel="nofollow noopener">$TSLA</a>`,
		`<a href="https://x.com/x" rel="nofollow noopener">https://x.com/x</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

// A hashtag glued to the end of a word is not a hashtag (foo#bar), and an @
// after a word char is part of an address, not a mention.
func TestLinkifyHTMLBoundaries(t *testing.T) {
	got := string(linkifyHTML("see foo#bar or me@example", x.Entities{}, nil))
	if strings.Contains(got, "<a") {
		t.Fatalf("mid-word # and @ should not link, got %q", got)
	}
}

// linkifyHTML escapes the surrounding text so a tweet body can never inject
// markup; only the anchors this package emits are live.
func TestLinkifyHTMLEscapesInjection(t *testing.T) {
	got := string(linkifyHTML(`<script>alert(1)</script>`, x.Entities{}, nil))
	if strings.Contains(got, "<script>") {
		t.Fatalf("raw markup leaked through: %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("angle brackets should be escaped, got %q", got)
	}
}

// Newlines are kept raw (the .text rule renders white-space: pre-wrap), so a
// line break is one newline, not a <br> on top of it.
func TestLinkifyHTMLKeepsNewlines(t *testing.T) {
	got := string(linkifyHTML("line one\nline two", x.Entities{}, nil))
	if strings.Contains(got, "<br>") {
		t.Fatalf("should not insert <br>; pre-wrap renders the newline, got %q", got)
	}
	if !strings.Contains(got, "line one\nline two") {
		t.Fatalf("newline should survive, got %q", got)
	}
}

func TestLinkifyMarkdown(t *testing.T) {
	got := LinkifyMarkdown("hey @jack #golang https://x.com/x.")
	for _, want := range []string{
		"[@jack](https://x.com/jack)",
		"[#golang](https://x.com/hashtag/golang)",
		"[https://x.com/x](https://x.com/x).",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	// The apostrophe-as-hashtag bug must not surface in Markdown either.
	md := LinkifyMarkdown(`it's done`)
	if strings.Contains(md, "hashtag") {
		t.Fatalf("apostrophe linked as a hashtag in markdown: %q", md)
	}
}
