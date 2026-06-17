package archive

import (
	"fmt"
	"time"

	"github.com/tamnd/tori/repo"
	"github.com/tamnd/tori/thread"
)

// RenderOptions controls a re-render of an existing repository.
type RenderOptions struct {
	Views   []string  // which shapes to write: "html", "md"
	Date    time.Time // footer stamp; zero means now
	Version string
}

// RenderResult summarises a re-render.
type RenderResult struct {
	Total   int
	Threads int
}

// Render re-renders the views of an existing repository from its stored JSON,
// touching no network (spec §5). It reads the records, the profile, and the
// manifest's media index (so localised media still resolves), reconstructs the
// threads, and writes the requested views over the existing ones.
func Render(root string, o RenderOptions) (*RenderResult, error) {
	st, err := repo.Open(root)
	if err != nil {
		return nil, err
	}
	mf, ok, err := repo.LoadManifest(root)
	if err != nil {
		return nil, err
	}
	if !ok || mf == nil {
		return nil, fmt.Errorf("%s is not a tori repository (no manifest.json)", root)
	}

	all, err := st.LoadTweets()
	if err != nil {
		return nil, err
	}
	profile, _, err := st.LoadProfile()
	if err != nil {
		return nil, err
	}
	threads := thread.Assemble(all)

	t := targetFromManifest(mf)
	stamp := o.Date
	if stamp.IsZero() {
		stamp = time.Now().UTC()
	}
	opts := Options{Views: o.Views, Date: stamp, Version: o.Version}

	if err := renderAll(st, all, threads, profile, mf.MediaIndex, t, opts); err != nil {
		return nil, err
	}
	return &RenderResult{Total: len(all), Threads: len(threads)}, nil
}

// targetFromManifest reconstructs the display target from a stored manifest so a
// re-render labels pages the same way the capture did.
func targetFromManifest(mf *repo.Manifest) Target {
	t := Target{Kind: Kind(mf.Target.Kind), Ref: mf.Target.Ref}
	switch t.Kind {
	case KindProfile:
		t.Display = "@" + t.Ref
	case KindSearch:
		t.Display = "Search: " + t.Ref
	case KindBookmarks:
		t.Display = "Bookmarks"
	case KindLikes:
		t.Display = "Likes of @" + t.Ref
	case KindList:
		t.Display = "List " + t.Ref
	case KindThread:
		t.Display = "Thread " + t.Ref
	case KindTweet:
		t.Display = "Post " + t.Ref
	default:
		t.Display = t.Ref
	}
	return t
}
