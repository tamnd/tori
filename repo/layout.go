// Package repo is the on-disk tori repository: the path-mapping rules, the
// record store, the manifest, and the incremental merge. layout.go is the pure
// heart of it — every file in a repository is a deterministic function of the
// record it holds, with no network and no clock, so the layout is testable in
// isolation and a re-capture lands on the exact same paths (spec §6).
package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"strings"
)

// Well-known files at the repository root.
const (
	ManifestFile = "manifest.json"
	StateFile    = "state.json"
	ProfileFile  = "profile.json"
	IndexFile    = "index.html"
	ReadmeFile   = "README.md"
	AssetsDir    = "_assets"
	CSSFile      = "_assets/tori.css"
)

// Sub-directories that hold the per-record files.
const (
	TweetsDir  = "tweets"
	ThreadsDir = "threads"
	HTMLDir    = "html"
	MDDir      = "md"
	MediaDir   = "media"
)

// TweetJSON is the canonical record path for a tweet id (spec §6.2). The id is a
// snowflake string used verbatim, so the path is a pure function of the id and a
// re-capture overwrites the same file.
func TweetJSON(id string) string { return path.Join(TweetsDir, id+".json") }

// TweetRaw is the untouched upstream payload beside the canonical record (TP3).
func TweetRaw(id string) string { return path.Join(TweetsDir, id+".raw.json") }

// TweetHTML is the rendered per-tweet page (kage-shape, inert).
func TweetHTML(id string) string { return path.Join(HTMLDir, id+".html") }

// TweetMD is the rendered per-tweet Markdown (yomi-shape).
func TweetMD(id string) string { return path.Join(MDDir, id+".md") }

// ThreadHTML is a reconstructed conversation rendered as one inert page.
func ThreadHTML(root string) string { return path.Join(ThreadsDir, root+".html") }

// ThreadMD is a reconstructed conversation rendered as one Markdown document.
func ThreadMD(root string) string { return path.Join(ThreadsDir, root+".md") }

// MediaSubdir is the directory a media type lives under within media/. X serves
// animated GIFs as mp4, so they get their own bucket distinct from real video.
func MediaSubdir(typ string) string {
	switch typ {
	case "photo":
		return "photo"
	case "video":
		return "video"
	case "animated_gif", "gif":
		return "gif"
	case "avatar":
		return "avatar"
	case "banner":
		return "banner"
	default:
		if typ == "" {
			return "other"
		}
		return safeSeg(typ)
	}
}

// MediaPath maps a media item to its deterministic local file (spec §6.3). The
// stem is the media key plus the first 6 hex of a sha256 of the source URL, so
// two renditions of one key never collide and a photo referenced by a thousand
// tweets resolves to one file. The extension comes from ext (derived from the
// URL or the Content-Type by the caller).
func MediaPath(typ, key, srcURL, ext string) string {
	sub := MediaSubdir(typ)
	stem := safeSeg(key) + "__" + shortHash(srcURL)
	return path.Join(MediaDir, sub, stem+normalizeExt(ext))
}

// AvatarPath maps a profile avatar to a stable local file keyed by handle and
// the size segment X embeds in its URL (e.g. _400x400).
func AvatarPath(handle, srcURL, ext string) string {
	size := avatarSize(srcURL)
	stem := safeSeg(handle)
	if size != "" {
		stem += "__" + size
	}
	return path.Join(MediaDir, MediaSubdir("avatar"), stem+normalizeExt(ext))
}

// BannerPath maps a profile banner to a stable local file keyed by handle.
func BannerPath(handle, ext string) string {
	stem := safeSeg(handle) + "__banner"
	return path.Join(MediaDir, MediaSubdir("banner"), stem+normalizeExt(ext))
}

// Rel returns the relative path from the directory holding the page at from to
// the file at to, both repository-relative with forward slashes (spec §6.4). It
// is what rewrites a media src or a cross-tweet link inside a rendered page so
// the archive resolves with the network unplugged. It never escapes the repo
// root because both inputs are already repo-relative.
func Rel(from, to string) string {
	from = path.Clean("/" + strings.ReplaceAll(from, "\\", "/"))
	to = path.Clean("/" + strings.ReplaceAll(to, "\\", "/"))
	fromDir := path.Dir(from)
	fseg := splitNonEmpty(fromDir)
	tseg := splitNonEmpty(to)
	// Drop the common leading directories.
	i := 0
	for i < len(fseg) && i < len(tseg) && fseg[i] == tseg[i] {
		i++
	}
	out := make([]string, 0, len(fseg)-i+len(tseg)-i)
	for j := i; j < len(fseg); j++ {
		out = append(out, "..")
	}
	out = append(out, tseg[i:]...)
	if len(out) == 0 {
		return "."
	}
	return strings.Join(out, "/")
}

// shortHash is the first 6 hex of a sha256 of s. Stable across runs and
// platforms, so it keeps the media filename deterministic (TP5).
func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:6]
}

// safeSeg makes an arbitrary identifier safe as one path segment: it keeps
// letters, digits, dot, underscore and hyphen, and replaces anything else with
// an underscore, so a key or handle can never escape its directory or inject a
// separator.
func safeSeg(s string) string {
	if s == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	// A leading dot or all-dots would be a hidden or traversal-ish name.
	out = strings.TrimLeft(out, ".")
	if out == "" {
		return "_"
	}
	return out
}

// normalizeExt ensures the extension begins with a dot, is lower-cased, and is a
// single safe segment; an empty ext yields no suffix.
func normalizeExt(ext string) string {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return "." + safeSeg(strings.ToLower(strings.TrimPrefix(ext, ".")))
}

// avatarSize extracts the X size segment (e.g. 400x400) from a profile-image URL
// like .../<id>/<name>_400x400.jpg, returning "" when none is present.
func avatarSize(srcURL string) string {
	base := path.Base(srcURL)
	if i := strings.LastIndex(base, "."); i >= 0 {
		base = base[:i]
	}
	if i := strings.LastIndex(base, "_"); i >= 0 {
		seg := base[i+1:]
		if strings.Contains(seg, "x") {
			return safeSeg(seg)
		}
	}
	return ""
}

func splitNonEmpty(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s != "" && s != "." {
			out = append(out, s)
		}
	}
	return out
}
