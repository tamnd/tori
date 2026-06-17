package archive

import (
	"fmt"
	"path"
	"strings"

	"github.com/tamnd/tori/repo"
	"github.com/tamnd/x-cli/x"
)

// Kind is what a capture target points at (spec §6.5, §7).
type Kind string

const (
	KindTweet     Kind = "tweet"
	KindThread    Kind = "thread"
	KindProfile   Kind = "profile"
	KindSearch    Kind = "search"
	KindBookmarks Kind = "bookmarks"
	KindLikes     Kind = "likes"
	KindList      Kind = "list"
)

// Target is a parsed, canonical capture target. Ref is the canonical identity
// used to both address the source and name the repository root: a profile's
// handle, a tweet/thread/list id, or a search query. Display is a human label
// for the page nav and headings.
type Target struct {
	Kind    Kind
	Ref     string
	Display string
}

// Selector carries the flags that pick a non-profile target kind, so a bare
// argument is read in the light of --search/--bookmarks/--likes/--list/--thread
// (spec §12.1). Exactly one of the kind-selecting fields may be set.
type Selector struct {
	Thread    bool
	Search    string
	Bookmarks bool
	Likes     string
	List      string
}

// ParseTarget resolves a CLI argument and the kind-selecting flags into a
// canonical Target. The grammar is x-cli's own ref grammar (TP via ParseTweetRef
// / ParseUserRef) plus tori's capture keywords: a status URL or numeric id is a
// tweet (a thread with --thread), a @handle or profile URL is a profile, and the
// --search/--bookmarks/--likes/--list flags select the timeline kinds. arg may
// be empty when a flag fully specifies the target (--bookmarks, --search).
func ParseTarget(arg string, sel Selector) (Target, error) {
	if err := sel.validate(); err != nil {
		return Target{}, err
	}
	switch {
	case sel.Bookmarks:
		return Target{Kind: KindBookmarks, Ref: "bookmarks", Display: "Bookmarks"}, nil
	case sel.Search != "":
		q := strings.TrimSpace(sel.Search)
		return Target{Kind: KindSearch, Ref: q, Display: "Search: " + q}, nil
	case sel.List != "":
		id := strings.TrimSpace(sel.List)
		return Target{Kind: KindList, Ref: id, Display: "List " + id}, nil
	case sel.Likes != "":
		ref, _, err := x.ParseUserRef(sel.Likes, false)
		if err != nil {
			return Target{}, fmt.Errorf("parse --likes user %q: %w", sel.Likes, err)
		}
		return Target{Kind: KindLikes, Ref: ref, Display: "Likes of @" + ref}, nil
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		return Target{}, fmt.Errorf("a target is required (a @handle, a tweet id or URL, or one of --search/--bookmarks/--likes/--list)")
	}

	// A status reference (URL with /status/, or a bare numeric id) is a tweet,
	// or a thread when --thread upgrades it.
	if id, err := x.ParseTweetRef(arg); err == nil {
		if sel.Thread {
			return Target{Kind: KindThread, Ref: id, Display: "Thread " + id}, nil
		}
		return Target{Kind: KindTweet, Ref: id, Display: "Post " + id}, nil
	}

	// Otherwise it is a profile handle or profile URL.
	ref, isID, err := x.ParseUserRef(arg, false)
	if err != nil {
		return Target{}, fmt.Errorf("parse target %q: %w", arg, err)
	}
	if isID {
		return Target{Kind: KindProfile, Ref: ref, Display: "User " + ref}, nil
	}
	return Target{Kind: KindProfile, Ref: ref, Display: "@" + ref}, nil
}

func (s Selector) validate() error {
	n := 0
	if s.Search != "" {
		n++
	}
	if s.Bookmarks {
		n++
	}
	if s.Likes != "" {
		n++
	}
	if s.List != "" {
		n++
	}
	if n > 1 {
		return fmt.Errorf("choose only one of --search/--bookmarks/--likes/--list")
	}
	return nil
}

// Root returns the repository directory for this target under out: out/x/<root>,
// where <root> is the canonical, filesystem-safe target identity (spec §6.1). Two
// captures of the same target land in the same repo and merge.
func (t Target) Root(out string) string {
	return path.Join(out, "x", t.slug())
}

// slug is the filesystem-safe root-directory name for the target.
func (t Target) slug() string {
	switch t.Kind {
	case KindProfile:
		return safeName(t.Ref)
	case KindTweet:
		return "status-" + safeName(t.Ref)
	case KindThread:
		return "thread-" + safeName(t.Ref)
	case KindSearch:
		return "search-" + safeName(t.Ref)
	case KindBookmarks:
		return "bookmarks"
	case KindLikes:
		return "likes-" + safeName(t.Ref)
	case KindList:
		return "list-" + safeName(t.Ref)
	default:
		return safeName(t.Ref)
	}
}

// TargetRef converts the parsed target into the manifest's record of what the
// repository archives.
func (t Target) TargetRef() repo.TargetRef {
	r := repo.TargetRef{Kind: string(t.Kind), Ref: t.Ref}
	if t.Kind == KindSearch {
		r.Query = t.Ref
	}
	return r
}

// safeName reduces an arbitrary ref to one safe, compact path segment.
func safeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_' || r == '-':
			b.WriteRune(r)
		case r == ' ' || r == ':' || r == '/' || r == '#':
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "capture"
	}
	if len(out) > 60 {
		out = strings.Trim(out[:60], "-")
	}
	return out
}
