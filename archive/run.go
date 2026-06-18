package archive

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/tamnd/tori/media"
	htmlrender "github.com/tamnd/tori/render/html"
	mdrender "github.com/tamnd/tori/render/md"
	"github.com/tamnd/tori/repo"
	"github.com/tamnd/tori/thread"
	"github.com/tamnd/x-cli/x"
)

// errBudget is the sentinel an emit callback returns to stop a paged read once
// the record budget (--max) is reached. It is not an error to the caller.
var errBudget = errors.New("record budget reached")

// errStopTimeline is the sentinel the hybrid timeline pass returns to stop paging
// once it crosses below the --since floor (the timeline is newest-first, so every
// later tweet is older). It ends pass 1 cleanly and is not an error to the caller.
var errStopTimeline = errors.New("timeline reached the since floor")

// Run captures a target into a repository under opts.Out and returns a summary.
// It is the one entry the CLI's archive/add commands call. The pipeline is:
// resolve and stream records into the store (writing each as it arrives so a
// session-limited or interrupted run keeps what it got), localise media, render
// the requested views, and write the manifest. Records always persist as JSON;
// media and views are derived and regenerable with `tori render`.
func Run(ctx context.Context, eng *x.Engine, t Target, opts Options, log Logf) (*Result, error) {
	root := t.Root(opts.Out)
	res := &Result{Root: root, Target: t}

	if opts.DryRun {
		res.DryRun = true
		say(log, "dry run: would capture %s into %s", t.Display, root)
		return res, nil
	}

	st, err := repo.Open(root)
	if err != nil {
		return res, err
	}

	mf, existed, err := repo.LoadManifest(root)
	if err != nil {
		return res, err
	}
	if !existed || mf == nil {
		mf = repo.NewManifest(t.TargetRef(), opts.Version)
	}
	mf.ToriVersion = opts.Version

	// Incremental bound: when adding to an existing repo, fetch only what is newer
	// than the newest tweet already held, unless --force or an explicit --since-id.
	sinceID := opts.SinceID
	if sinceID == "" && existed && !opts.Force {
		if held, _ := st.LoadTweets(); len(held) > 0 {
			sinceID = newestID(held)
		}
	}

	cap := &capturer{
		st:      st,
		opts:    opts,
		log:     log,
		sinceID: sinceID,
		seen:    map[string]bool{},
	}

	profile, err := cap.capture(ctx, eng, t, res)
	// A budget stop or a context cancel is not a failure: keep what was written.
	if err != nil && !errors.Is(err, errBudget) && ctx.Err() == nil {
		// Surface the typed error to the CLI for the right exit code, but only
		// after persisting whatever the store already holds below.
		return res, finishErr(res, st, mf, profile, log, err)
	}
	res.Profile = profile

	// Re-read the full record set (new plus previously held) so media and render
	// operate on the merged archive (spec §11).
	all, err := st.LoadTweets()
	if err != nil {
		return res, err
	}
	res.Total = len(all)
	threads := thread.Assemble(all)
	res.Threads = len(threads)
	setRange(res, all)

	// Localise media through the shared client (TP1).
	assets := mf.MediaIndex
	if opts.Media != media.PolicyNone {
		items := media.Plan(profile, all, opts.Media, opts.Video)
		say(log, "media: %d items planned", len(items))
		mr := media.Download(ctx, eng.Client(), st, items, func(f string, a ...any) { say(log, f, a...) })
		res.MediaOK = mr.Downloaded + mr.Reused
		res.MediaFail = mr.Failed
		res.StreamOnly = mr.StreamOnly
		assets = mergeAssets(assets, mr.Assets)
	}

	// Render the requested views from the stored records (TP3).
	if err := renderAll(st, all, threads, profile, assets, t, opts); err != nil {
		return res, err
	}

	// Manifest: counts, range, media index, and a capture entry (the only
	// wall-clock value, TP5).
	mf.Tweets = len(all)
	mf.Threads = len(threads)
	mf.Media = countLocal(assets)
	mf.MediaIndex = assets
	mf.Range = repoRange(all)
	if profile != nil && profile.ID != "" {
		mf.Target.UserID = profile.ID
	}
	mf.AddCapture(opts.stamp().Format(time.RFC3339), res.Added, primaryTier(eng))
	if err := mf.Save(root); err != nil {
		return res, err
	}
	res.Tiers = mf.TiersUsed

	if ctx.Err() != nil {
		return res, ctx.Err()
	}
	return res, nil
}

// finishErr persists the partial archive before returning a fatal capture error,
// so a needs-auth or rate-limit stop still leaves a valid, smaller repo on disk.
func finishErr(res *Result, st *repo.Store, mf *repo.Manifest, profile *x.User, log Logf, cause error) error {
	all, err := st.LoadTweets()
	if err != nil {
		return cause
	}
	mf.Tweets = len(all)
	mf.Range = repoRange(all)
	if profile != nil && profile.ID != "" {
		mf.Target.UserID = profile.ID
	}
	_ = mf.Save(res.Root)
	say(log, "partial capture saved: %d tweets", len(all))
	return cause
}

// capturer holds the per-run capture state shared across the kind handlers.
type capturer struct {
	st      *repo.Store
	opts    Options
	log     Logf
	sinceID string
	seen    map[string]bool
	count   int // records emitted (post-filter) this run, for the --max budget
}

// capture dispatches on the target kind, streaming records into the store and
// returning the captured profile when the target has one.
func (c *capturer) capture(ctx context.Context, eng *x.Engine, t Target, res *Result) (*x.User, error) {
	switch t.Kind {
	case KindTweet:
		return nil, c.captureTweet(ctx, eng, t.Ref, res)
	case KindThread:
		return nil, c.captureThread(ctx, eng, t.Ref, res)
	case KindProfile:
		return c.captureProfile(ctx, eng, t.Ref, res)
	case KindSearch:
		return nil, c.captureSearch(ctx, eng, t.Ref, res)
	case KindBookmarks:
		return nil, c.captureStream(res, func(emit func(*x.Tweet) error) error {
			return eng.GraphQL().Bookmarks(ctx, c.opts.Max, emit)
		}, true)
	case KindLikes:
		return nil, c.captureStream(res, func(emit func(*x.Tweet) error) error {
			return eng.Likes(ctx, t.Ref, false, c.opts.Max, emit)
		}, true)
	case KindList:
		return nil, c.captureStream(res, func(emit func(*x.Tweet) error) error {
			return eng.GraphQL().ListTweets(ctx, t.Ref, c.opts.Max, emit)
		}, true)
	default:
		return nil, fmt.Errorf("unsupported target kind %q", t.Kind)
	}
}

// emit returns the per-record callback: it dedupes, applies the timeline shape
// filter when shaping is on, writes the record immediately, counts new tweets,
// and trips the budget. A fresh tweet (not already on disk) increments Added.
func (c *capturer) emit(res *Result, shape bool) func(*x.Tweet) error {
	return func(t *x.Tweet) error {
		if t == nil || t.ID == "" {
			return nil
		}
		if c.seen[t.ID] {
			return nil
		}
		c.seen[t.ID] = true
		if shape && !c.keep(t) {
			return nil
		}
		fresh := !c.st.HasTweet(t.ID)
		if err := c.st.WriteTweet(t, nil); err != nil {
			return err
		}
		if fresh {
			res.Added++
		}
		c.count++
		if c.opts.Verbose {
			say(c.log, "  %s %s", t.ID, oneLine(t.Text))
		}
		if c.opts.Max > 0 && c.count >= c.opts.Max {
			return errBudget
		}
		return nil
	}
}

// keep applies the timeline-shaping flags and bounds to one tweet.
func (c *capturer) keep(t *x.Tweet) bool {
	o := c.opts
	if o.MediaOnly && len(t.Media) == 0 {
		return false
	}
	if !o.WithReplies && t.IsReply {
		return false
	}
	if !o.WithRetweets && t.IsRetweet {
		return false
	}
	if !o.Since.IsZero() && t.CreatedAt.Before(o.Since) {
		return false
	}
	if !o.Until.IsZero() && !t.CreatedAt.Before(o.Until) {
		return false
	}
	if c.sinceID != "" && !idNewer(t.ID, c.sinceID) {
		return false
	}
	if o.UntilID != "" && idNewer(t.ID, o.UntilID) {
		return false
	}
	return true
}

func (c *capturer) captureTweet(ctx context.Context, eng *x.Engine, id string, res *Result) error {
	t, err := eng.Tweet(ctx, id)
	if err != nil {
		return err
	}
	return c.emit(res, false)(t)
}

func (c *capturer) captureThread(ctx context.Context, eng *x.Engine, id string, res *Result) error {
	emit := c.emit(res, false)
	return eng.Thread(ctx, id, c.opts.Max, emit)
}

func (c *capturer) captureSearch(ctx context.Context, eng *x.Engine, q string, res *Result) error {
	emit := c.emit(res, true)
	return eng.Search(ctx, x.SearchQuery{Raw: q, Product: "Latest", Limit: c.opts.Max}, emit)
}

// captureStream runs an arbitrary paged reader (bookmarks/likes/list) under the
// shape filter and budget.
func (c *capturer) captureStream(res *Result, read func(emit func(*x.Tweet) error) error, shape bool) error {
	return read(c.emit(res, shape))
}

func (c *capturer) captureProfile(ctx context.Context, eng *x.Engine, ref string, res *Result) (*x.User, error) {
	u, err := eng.User(ctx, ref, false)
	if err != nil {
		return nil, err
	}
	if u != nil {
		if err := c.st.WriteProfile(u); err != nil {
			return u, err
		}
		say(c.log, "profile @%s: %s, %d posts", u.Username, u.Name, u.Metrics.Tweets)
	}

	emit := c.emit(res, true)

	if c.opts.ByMonth {
		// Full history, hybrid two-pass (spec §7.1). A user timeline caps at roughly
		// 3200 tweets, so the only free way to the whole history is month-wide
		// `from:<handle> since:.. until:..` search windows. But search is the scarce
		// quota. So pass 1 streams the timeline (UserTweets/UserTweetsAndReplies),
		// which lives on a *separate* rate-limit quota, to grab the dense recent
		// window cheaply; pass 2 then walks search windows only for the older gap the
		// timeline could not reach. The shared emit/seen set dedupes the overlap.
		handle := ref
		if u != nil && u.Username != "" {
			handle = u.Username
		}
		from := time.Date(2006, 7, 1, 0, 0, 0, 0, time.UTC) // X launch, the earliest possible
		if u != nil && !u.CreatedAt.IsZero() {
			from = u.CreatedAt
		}
		// --since raises the floor: a bounded by-month run walks only the windows
		// it needs, which also keeps a free-tier run under the search rate limit.
		if !c.opts.Since.IsZero() && c.opts.Since.After(from) {
			from = c.opts.Since
		}
		to := time.Now().UTC()
		if !c.opts.Until.IsZero() {
			to = c.opts.Until
		}

		// Pass 1: the timeline. It pages off a different quota than search, so this
		// load does not eat into the windows below.
		fullTo := to
		tlStart := time.Now()
		before := res.Added
		oldest, tlErr := c.timelinePass(ctx, eng, ref, res)
		switch {
		case errors.Is(tlErr, errBudget):
			return u, tlErr
		case ctx.Err() != nil:
			return u, ctx.Err()
		case tlErr != nil && !errors.Is(tlErr, errStopTimeline):
			// The timeline endpoint failed outright; do not trust its boundary, and
			// fall back to walking the whole range on search alone.
			say(c.log, "timeline pass: %v (search will cover the full range)", tlErr)
			oldest = time.Time{}
		}
		if !oldest.IsZero() {
			say(c.log, "timeline pass: +%d, reached %s in %s", res.Added-before, oldest.Format("2006-01-02"), time.Since(tlStart).Round(time.Second))
			// Search only needs the gap older than what the timeline reached.
			if oldest.Before(to) {
				to = oldest
			}
		}

		// Tier 0 has no search; the timeline window is all it can offer.
		if !eng.CanGraphQL() {
			res.Note("Tier 0 cannot run search windows; --by-month needs --guest or a session. Captured only the recent timeline window.")
			return u, nil
		}

		// Pass 2: search the remaining older windows.
		windows := monthWindows(from, to)
		full := len(monthWindows(from, fullTo))
		say(c.log, "by-month: %d/%d windows from %s (timeline covered the %d newest)", len(windows), full, from.Format("2006-01"), full-len(windows))
		for _, w := range windows {
			if ctx.Err() != nil {
				return u, ctx.Err()
			}
			q := fmt.Sprintf("from:%s since:%s until:%s", handle, w[0].Format("2006-01-02"), w[1].Format("2006-01-02"))
			before := res.Added
			err := eng.Search(ctx, x.SearchQuery{Raw: q, Product: "Latest", Limit: c.opts.Max}, emit)
			res.Windows++
			if errors.Is(err, errBudget) {
				return u, err
			}
			if err != nil {
				// One bad window should not abort the whole history; record and move on.
				say(c.log, "  window %s: %v", w[0].Format("2006-01"), err)
				continue
			}
			if added := res.Added - before; added > 0 {
				say(c.log, "  %s: +%d", w[0].Format("2006-01"), added)
			}
		}
		return u, nil
	}

	// Default: stream the timeline. Tier 0 gives the recent window; a GraphQL tier
	// pages deeper. The engine picks the cheapest surface.
	o := x.TimelineOpts{Replies: c.opts.WithReplies, Media: c.opts.MediaOnly, Limit: c.opts.Max}
	err = eng.Timeline(ctx, ref, false, o, emit)
	if err == nil && !eng.CanGraphQL() {
		res.Note("Tier 0 returned only the recent timeline window; pass --guest to page deeper, or --by-month for full history.")
	}
	return u, err
}

// timelinePass is pass 1 of the hybrid full-history capture: it streams the user
// timeline into the store through the shared emit and returns the oldest tweet
// timestamp it reached, which becomes the upper bound for the search windows.
// Because the timeline endpoint sits on a different rate-limit quota than search,
// this captures the dense recent window without spending the search budget the
// older windows need. It stops early once it pages below the --since floor.
func (c *capturer) timelinePass(ctx context.Context, eng *x.Engine, ref string, res *Result) (time.Time, error) {
	var oldest time.Time
	emit := c.emit(res, true)
	track := func(t *x.Tweet) error {
		if t != nil && !t.CreatedAt.IsZero() {
			if oldest.IsZero() || t.CreatedAt.Before(oldest) {
				oldest = t.CreatedAt
			}
			// Newest-first: once past the --since floor every later tweet is older
			// too, so stop here and let the (empty) search gap close the run.
			if !c.opts.Since.IsZero() && t.CreatedAt.Before(c.opts.Since) {
				return errStopTimeline
			}
		}
		return emit(t)
	}
	o := x.TimelineOpts{Replies: c.opts.WithReplies, Media: c.opts.MediaOnly, Limit: c.opts.Max}
	return oldest, eng.Timeline(ctx, ref, false, o, track)
}

// monthWindows returns [start,end) month spans from `to` back to `from`,
// newest-first, each aligned to the first of the month in UTC.
func monthWindows(from, to time.Time) [][2]time.Time {
	from = from.UTC()
	to = to.UTC()
	cur := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0) // include the current month fully
	floor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	var out [][2]time.Time
	for cur.After(floor) {
		start := cur.AddDate(0, -1, 0)
		out = append(out, [2]time.Time{start, cur})
		cur = start
	}
	return out
}

// renderAll writes the requested views plus the shared CSS.
func renderAll(st *repo.Store, all []*x.Tweet, threads []thread.Thread, profile *x.User, assets []repo.Asset, t Target, opts Options) error {
	footer := "Archived with tori · captured " + opts.stamp().Format("2006-01-02 15:04 MST")
	heading, sub := indexLabels(t, profile)

	if opts.wantHTML() {
		if err := st.WriteMedia(repo.CSSFile, htmlrender.CSS()); err != nil {
			return err
		}
		hr := htmlrender.New(all, assets, profile, footer, t.Display)
		page, err := hr.Index(threads, heading, sub)
		if err != nil {
			return err
		}
		if err := st.WriteText(repo.IndexFile, page); err != nil {
			return err
		}
		for _, th := range threads {
			if th.Standalone() {
				p, err := hr.TweetPage(th.Root)
				if err != nil {
					return err
				}
				if err := st.WriteText(repo.TweetHTML(th.RootID), p); err != nil {
					return err
				}
				continue
			}
			p, err := hr.ThreadPage(th)
			if err != nil {
				return err
			}
			if err := st.WriteText(repo.ThreadHTML(th.RootID), p); err != nil {
				return err
			}
		}
	}

	if opts.wantMD() {
		mr := mdrender.New(all, assets, profile, footer, t.Display)
		if err := st.WriteText(repo.ReadmeFile, mr.Index(threads, heading, sub)); err != nil {
			return err
		}
		for _, th := range threads {
			if th.Standalone() {
				if err := st.WriteText(repo.TweetMD(th.RootID), mr.Tweet(th.Root)); err != nil {
					return err
				}
				continue
			}
			if err := st.WriteText(repo.ThreadMD(th.RootID), mr.Thread(th)); err != nil {
				return err
			}
		}
	}
	return nil
}

// indexLabels returns the index heading/subheading for a non-profile capture;
// for a profile the header comes from profile.json instead, so both are empty.
func indexLabels(t Target, profile *x.User) (string, string) {
	if profile != nil {
		return "", ""
	}
	switch t.Kind {
	case KindSearch:
		return "Search", t.Ref
	case KindBookmarks:
		return "Bookmarks", ""
	case KindLikes:
		return "Likes", "@" + t.Ref
	case KindList:
		return "List", t.Ref
	default:
		// Tweet and thread index pages need no heading; the single card carries it.
		return "", ""
	}
}

// newestID returns the highest (newest) tweet id in a set.
func newestID(tweets []*x.Tweet) string {
	best := ""
	for _, t := range tweets {
		if best == "" || idNewer(t.ID, best) {
			best = t.ID
		}
	}
	return best
}

// idNewer reports whether id a is newer (numerically larger) than b, comparing
// snowflake strings by length then lexically.
func idNewer(a, b string) bool {
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	return a > b
}

func setRange(res *Result, all []*x.Tweet) {
	r := repoRange(all)
	if !r.Oldest.IsZero() {
		res.Oldest = r.Oldest.UTC().Format(time.RFC3339)
	}
	if !r.Newest.IsZero() {
		res.Newest = r.Newest.UTC().Format(time.RFC3339)
	}
}

func repoRange(all []*x.Tweet) repo.Range {
	var r repo.Range
	for _, t := range all {
		if t.CreatedAt.IsZero() {
			continue
		}
		if r.Oldest.IsZero() || t.CreatedAt.Before(r.Oldest) {
			r.Oldest = t.CreatedAt
		}
		if r.Newest.IsZero() || t.CreatedAt.After(r.Newest) {
			r.Newest = t.CreatedAt
		}
	}
	return r
}

// mergeAssets unions previously held assets with this run's, keyed by source URL,
// preferring the fresher record. The result is sorted for a stable manifest.
func mergeAssets(old, fresh []repo.Asset) []repo.Asset {
	by := map[string]repo.Asset{}
	for _, a := range old {
		by[a.Source] = a
	}
	for _, a := range fresh {
		by[a.Source] = a
	}
	out := make([]repo.Asset, 0, len(by))
	for _, a := range by {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		return out[i].Source < out[j].Source
	})
	return out
}

func countLocal(assets []repo.Asset) int {
	n := 0
	for _, a := range assets {
		if a.Status == "local" {
			n++
		}
	}
	return n
}

// primaryTier reports the tier the engine resolved to, for the capture record.
func primaryTier(eng *x.Engine) string {
	cfg := eng.Config()
	switch {
	case cfg.HasSession():
		return "session"
	case cfg.AllowGuest || cfg.Tier == "guest":
		return "guest"
	default:
		return "syndication"
	}
}

func oneLine(s string) string {
	for i, r := range s {
		if r == '\n' || i >= 60 {
			return s[:i] + "…"
		}
	}
	return s
}
