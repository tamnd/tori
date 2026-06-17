// Package media localises a record set's media: it collects every distinct
// reference, downloads each through the shared x.Client (so media rides the one
// rate limiter and cache), picks a video rendition, and maps each item to a
// deterministic local file. name.go and plan.go are pure (no network); only
// download.go touches the wire.
package media

import (
	"path"
	"strings"

	"github.com/tamnd/x-cli/x"
)

// Policy selects which media a capture localises (spec §8).
type Policy string

const (
	PolicyAll    Policy = "all"    // photos, video, gif, avatars, banners
	PolicyPhotos Policy = "photos" // photos + avatars + banners, no video/gif
	PolicyNone   Policy = "none"   // text-only, download nothing
)

// VideoPref selects a rendition for video and gif.
type VideoPref string

const (
	VideoBest  VideoPref = "best"
	VideoWorst VideoPref = "worst"
)

// Item is one media reference to fetch and where it will land. The Path is the
// deterministic repo-relative destination; Kind classifies it for the manifest.
type Item struct {
	Key    string // media key, or a derived key for avatar/banner
	Type   string // photo|video|gif|avatar|banner
	Source string // the URL to fetch (the picked rendition for video/gif)
	Path   string // repo-relative destination (empty until named)
	AltErr string // when non-empty, why this item cannot be localised (e.g. stream-only)
}

// pickVariant chooses a video/gif rendition URL by preference. X serves a list
// of mp4 variants at different bitrates plus, sometimes, an HLS/DASH master with
// no bitrate; those streams are not progressively downloadable, so an item with
// only stream variants returns ok=false with the master URL for the manifest.
func pickVariant(m x.Media, pref VideoPref) (url string, streamOnly bool) {
	type cand struct {
		url     string
		bitrate int
	}
	var mp4s []cand
	var stream string
	for _, v := range m.Variants {
		switch {
		case strings.Contains(v.ContentType, "mp4"):
			mp4s = append(mp4s, cand{v.URL, v.Bitrate})
		case strings.Contains(v.ContentType, "mpegurl") || strings.Contains(v.ContentType, "dash"):
			if stream == "" {
				stream = v.URL
			}
		}
	}
	if len(mp4s) == 0 {
		// No progressive rendition; fall back to the media URL if it is an mp4,
		// else report stream-only.
		if strings.Contains(m.URL, ".mp4") {
			return m.URL, false
		}
		if stream == "" {
			stream = m.URL
		}
		return stream, true
	}
	best := mp4s[0]
	for _, c := range mp4s[1:] {
		if pref == VideoWorst {
			if c.bitrate < best.bitrate {
				best = c
			}
		} else { // best
			if c.bitrate > best.bitrate {
				best = c
			}
		}
	}
	return best.url, false
}

// collectExt derives a file extension from a URL path, stripping any query, or
// "" when the URL carries none.
func collectExt(rawURL string) string {
	u := rawURL
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	ext := path.Ext(u)
	return strings.TrimPrefix(ext, ".")
}

// photoExt returns a usable extension for a photo URL, defaulting to jpg.
func photoExt(rawURL string) string {
	if e := collectExt(rawURL); e != "" {
		return e
	}
	return "jpg"
}
