package repo

import "testing"

func TestRel(t *testing.T) {
	cases := []struct {
		from, to, want string
	}{
		// Index links down into html/ and the assets dir.
		{"index.html", "html/123.html", "html/123.html"},
		{"index.html", "_assets/tori.css", "_assets/tori.css"},
		// A per-tweet page links back up to the css and across to media.
		{"html/123.html", "_assets/tori.css", "../_assets/tori.css"},
		{"html/123.html", "media/photo/abc.jpg", "../media/photo/abc.jpg"},
		// Two pages in the same directory are siblings.
		{"html/123.html", "html/456.html", "456.html"},
		// A path to its own file resolves to the bare name.
		{"html/123.html", "html/123.html", "123.html"},
	}
	for _, c := range cases {
		if got := Rel(c.from, c.to); got != c.want {
			t.Errorf("Rel(%q, %q) = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}

// A relative link never escapes the repository root with stray ../ segments.
func TestRelStaysInRepo(t *testing.T) {
	got := Rel("media/photo/abc.jpg", "index.html")
	if got != "../../index.html" {
		t.Errorf("Rel = %q, want ../../index.html", got)
	}
}

func TestRecordPathsAreDeterministic(t *testing.T) {
	if TweetJSON("20") != "tweets/20.json" {
		t.Errorf("TweetJSON = %q", TweetJSON("20"))
	}
	if TweetHTML("20") != "html/20.html" {
		t.Errorf("TweetHTML = %q", TweetHTML("20"))
	}
	if ThreadMD("20") != "threads/20.md" {
		t.Errorf("ThreadMD = %q", ThreadMD("20"))
	}
}
