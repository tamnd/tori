package html

import (
	"strings"
	"testing"
	"time"

	"github.com/tamnd/tori/thread"
	"github.com/tamnd/x-cli/x"
)

func sampleTweet() *x.Tweet {
	return &x.Tweet{
		ID:             "200",
		ConversationID: "200",
		URL:            "https://x.com/jack/status/200",
		Text:           "it's a #test with @jack\nsecond line",
		CreatedAt:      time.Date(2025, 1, 2, 3, 4, 0, 0, time.UTC),
		Author:         &x.User{Username: "jack", Name: "Jack"},
	}
}

// The rendered page must carry the linkified anchors verbatim, not a second time
// HTML-escaped (bug B: HTMLBody was a string and html/template escaped it again,
// so <a> showed up as &lt;a&gt; and #39 from an apostrophe became a hashtag).
func TestTweetPageEmitsLinkifiedHTMLOnce(t *testing.T) {
	r := New([]*x.Tweet{sampleTweet()}, nil, nil, "footer", "@jack")
	page, err := r.TweetPage(sampleTweet())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(page, "&lt;a ") || strings.Contains(page, "&amp;lt;") {
		t.Fatalf("anchor markup was double-escaped: %s", excerpt(page))
	}
	if !strings.Contains(page, `<a href="https://x.com/hashtag/test"`) {
		t.Fatalf("hashtag not linkified in page: %s", excerpt(page))
	}
	if !strings.Contains(page, `<a href="https://x.com/jack"`) {
		t.Fatalf("mention not linkified in page: %s", excerpt(page))
	}
	// The apostrophe stays escaped text, never a hashtag link.
	if strings.Contains(page, "hashtag/39") {
		t.Fatalf("apostrophe linked as hashtag: %s", excerpt(page))
	}
	if !strings.Contains(page, "it&#39;s") {
		t.Fatalf("apostrophe should be escaped text: %s", excerpt(page))
	}
}

// The index links each card to its local page and offers a source link to X, so
// the archive is navigable offline (spec §9).
func TestIndexLinksLocalAndSource(t *testing.T) {
	tw := sampleTweet()
	r := New([]*x.Tweet{tw}, nil, nil, "footer", "@jack")
	page, err := r.Index(thread.Assemble([]*x.Tweet{tw}), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, `href="html/200.html"`) {
		t.Fatalf("index should link to the local per-tweet page: %s", excerpt(page))
	}
	if !strings.Contains(page, `href="https://x.com/jack/status/200"`) {
		t.Fatalf("index should keep a source link to X: %s", excerpt(page))
	}
}

// The page is inert: no script tags and no inline event handlers.
func TestPageIsInert(t *testing.T) {
	r := New([]*x.Tweet{sampleTweet()}, nil, nil, "footer", "@jack")
	page, _ := r.TweetPage(sampleTweet())
	if strings.Contains(strings.ToLower(page), "<script") {
		t.Error("rendered page contains a script tag")
	}
	if strings.Contains(strings.ToLower(page), " onclick") || strings.Contains(strings.ToLower(page), " onload") {
		t.Error("rendered page contains an inline event handler")
	}
}

func excerpt(s string) string {
	if len(s) > 400 {
		return s[:400]
	}
	return s
}
