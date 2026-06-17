// Package archive is the capture pipeline that turns a target into a tori
// repository: it plans what to fetch on the cheapest free tier, streams the
// records through the x-cli engine into the store, localises the media, renders
// the HTML and Markdown views, and writes the manifest (spec §5, §7). It owns no
// scraping of its own — every byte from X comes through the x-cli engine (TP1) —
// and no presentation — every page comes through render/ (TP3). What it adds is
// the artifact: a self-contained, deterministic, resumable archive.
package archive

import (
	"time"

	"github.com/tamnd/tori/media"
)

// Options is the resolved configuration for one capture, built from the CLI
// flags. The zero value is not valid; the CLI fills it. Fetch-shaping fields
// mirror the x-cli timeline/search readers so a tori flag maps straight onto an
// engine call (spec §12.1).
type Options struct {
	// Output.
	Out   string   // output root; the repo lands at <Out>/x/<root>
	Views []string // which rendered shapes to write: "html", "md"

	// Media.
	Media     media.Policy    // all | photos | none
	Video     media.VideoPref // best | worst
	VideoKbps int             // when >0, pick the rendition nearest this bitrate (overrides Video)
	Tool      string          // optional external downloader for stream-only video (e.g. yt-dlp)

	// Record shaping.
	Max          int  // hard record budget; 0 means as many as the tier gives
	Thread       bool // upgrade a tweet target into a full-conversation capture
	WithReplies  bool // include replies in a profile/timeline capture
	WithRetweets bool // include retweets in a profile/timeline capture
	MediaOnly    bool // only tweets carrying media

	// Timeline bounds.
	Since   time.Time // only tweets at or after this time
	Until   time.Time // only tweets before this time
	SinceID string    // only tweets newer than this id
	UntilID string    // only tweets older than this id

	// Full-history exhaustion. A user timeline caps at roughly 3200 tweets, so to
	// archive an account's whole history tori walks month-wide search windows
	// (from:<handle> since:.. until:..) from the account's creation to now,
	// stitching the results into one timeline (spec §7.1). Requires a GraphQL tier.
	ByMonth bool

	// Link chasing (bounded), via x.Walk.
	Depth  int
	Fanout int

	// Run control.
	Date    time.Time // capture stamp written into the manifest (TP5); zero means now
	Resume  bool      // continue from state.json when present (default on)
	Force   bool      // ignore held state and re-capture from scratch
	DryRun  bool      // plan only; fetch and write nothing
	Verbose bool      // log the tier and endpoint per record
	Version string    // tori build version, recorded in the manifest
}

// wantHTML reports whether the HTML view should be rendered.
func (o Options) wantHTML() bool { return o.hasView("html") }

// wantMD reports whether the Markdown view should be rendered.
func (o Options) wantMD() bool { return o.hasView("md") }

func (o Options) hasView(v string) bool {
	for _, x := range o.Views {
		if x == v {
			return true
		}
	}
	return false
}

// stamp returns the capture timestamp to record, defaulting to now when unset.
func (o Options) stamp() time.Time {
	if o.Date.IsZero() {
		return time.Now().UTC()
	}
	return o.Date.UTC()
}
