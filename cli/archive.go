package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/tori/archive"
	"github.com/tamnd/tori/media"
)

// newArchiveCmd builds either `tori archive` or, when add is true, `tori add`.
// They share the same capture machinery; add defaults to the incremental path
// (fetch only what is new) and is just the friendlier name for a re-run.
func newArchiveCmd(add bool) *cobra.Command {
	use := "archive <target>..."
	short := "Capture a target into a new or existing repository"
	aliases := []string(nil)
	if add {
		use = "add <target>..."
		short = "Fetch only what is new for an existing target and re-render"
		aliases = []string{"update"}
	}

	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Example: archiveExamples,
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(cmd, args)
		},
	}

	f := cmd.Flags()
	// Target selectors.
	f.Bool("thread", false, "capture the whole conversation rooted at a tweet target")
	f.String("search", "", "capture a search query instead of a profile")
	f.Bool("bookmarks", false, "capture your own bookmarks (needs an imported session)")
	f.String("likes", "", "capture the tweets a user liked")
	f.String("list", "", "capture a List's timeline by id")

	// Record shaping.
	f.Bool("with-replies", false, "include replies in a profile or timeline capture")
	f.Bool("with-retweets", false, "include retweets in a profile or timeline capture")
	f.Bool("media-only", false, "capture only tweets that carry media")
	f.Bool("by-month", false, "exhaust a profile's full history via monthly search windows (needs --guest or a session)")
	f.String("since", "", "only tweets at or after this time (RFC3339 or 2006-01-02)")
	f.String("until", "", "only tweets before this time (RFC3339 or 2006-01-02)")
	f.String("since-id", "", "only tweets newer than this id")
	f.String("until-id", "", "only tweets older than this id")
	f.Int("max", 0, "record budget (0 = as many as the tier gives; default 1000 for a profile/search)")

	// Media.
	f.String("media", "all", "media to localise: all|photos|none")
	f.String("video", "best", "video rendition: best|worst")
	f.String("tool", "", "external downloader for stream-only video (e.g. yt-dlp)")

	// Output and rendering.
	f.String("view", "html,md", "views to render: html|md|html,md (JSON is always written)")
	f.StringP("out", "o", defaultOut(), "output root; the repo lands at <out>/x/<root>")
	f.String("date", "", "fix the capture stamp (RFC3339) for reproducible output")
	f.Bool("force", false, "ignore held state and recapture from scratch")
	f.Bool("dry-run", false, "print what would be captured without fetching")

	return cmd
}

const archiveExamples = `  tori archive 20                                   # one tweet, no setup
  tori archive https://x.com/jack/status/20 --thread
  tori archive karpathy                             # profile + recent timeline + media (Tier 0)
  tori archive karpathy --guest --by-month          # full history via monthly windows
  tori archive karpathy --guest --max 2000          # page deeper on the free guest tier
  tori archive --search "from:nasa #Artemis" --guest
  tori archive --bookmarks                          # your bookmarks (needs imported session)
  tori archive karpathy --view html,md -o ~/archives`

func runArchive(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()

	sel := archive.Selector{}
	sel.Thread, _ = f.GetBool("thread")
	sel.Search, _ = f.GetString("search")
	sel.Bookmarks, _ = f.GetBool("bookmarks")
	sel.Likes, _ = f.GetString("likes")
	sel.List, _ = f.GetString("list")

	// Build the option set once; it applies to every target in this run.
	opts, err := optionsFromFlags(cmd)
	if err != nil {
		return err
	}

	// Determine the targets: either the positional args, or a single
	// flag-specified target (search/bookmarks) with no positional.
	var rawTargets []string
	if len(args) > 0 {
		rawTargets = args
	} else {
		rawTargets = []string{""}
	}

	eng := engineFromFlags(cmd)
	log := stderrLog(cmd)
	ctx := cmd.Context()

	var firstErr error
	had := false
	for _, raw := range rawTargets {
		t, err := archive.ParseTarget(raw, sel)
		if err != nil {
			return err // a malformed target is a usage error
		}
		had = true
		res, err := archive.Run(ctx, eng, t, opts, log)
		printResult(cmd, res)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		// Only one flag-specified target makes sense per run.
		if sel.Search != "" || sel.Bookmarks || sel.Likes != "" || sel.List != "" {
			break
		}
	}
	if !had {
		return fmt.Errorf("nothing to capture")
	}
	return firstErr
}

func optionsFromFlags(cmd *cobra.Command) (archive.Options, error) {
	f := cmd.Flags()
	var o archive.Options
	o.Version = Version

	o.Out, _ = f.GetString("out")
	o.WithReplies, _ = f.GetBool("with-replies")
	o.WithRetweets, _ = f.GetBool("with-retweets")
	o.MediaOnly, _ = f.GetBool("media-only")
	o.ByMonth, _ = f.GetBool("by-month")
	o.Max, _ = f.GetInt("max")
	o.SinceID, _ = f.GetString("since-id")
	o.UntilID, _ = f.GetString("until-id")
	o.Force, _ = f.GetBool("force")
	o.DryRun, _ = f.GetBool("dry-run")
	o.Verbose, _ = f.GetBool("verbose")
	o.Tool, _ = f.GetString("tool")

	since, _ := f.GetString("since")
	if t, err := parseLooseTime(since); err != nil {
		return o, fmt.Errorf("parse --since: %w", err)
	} else {
		o.Since = t
	}
	until, _ := f.GetString("until")
	if t, err := parseLooseTime(until); err != nil {
		return o, fmt.Errorf("parse --until: %w", err)
	} else {
		o.Until = t
	}
	dateStr, _ := f.GetString("date")
	if t, err := parseDate(dateStr); err != nil {
		return o, fmt.Errorf("parse --date: %w", err)
	} else {
		o.Date = t
	}

	switch m, _ := f.GetString("media"); media.Policy(m) {
	case media.PolicyAll, media.PolicyPhotos, media.PolicyNone:
		o.Media = media.Policy(m)
	default:
		return o, fmt.Errorf("invalid --media %q (want all|photos|none)", m)
	}
	switch v, _ := f.GetString("video"); media.VideoPref(v) {
	case media.VideoBest, media.VideoWorst:
		o.Video = media.VideoPref(v)
	default:
		return o, fmt.Errorf("invalid --video %q (want best|worst)", v)
	}

	views, err := parseViews(mustString(f, "view"))
	if err != nil {
		return o, err
	}
	o.Views = views

	// Default record budget for an unbounded timeline kind, so a bare
	// `tori archive <handle>` does not try to pull a whole history by accident.
	if o.Max == 0 && !o.ByMonth {
		o.Max = 1000
	}
	return o, nil
}

func parseViews(s string) ([]string, error) {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(strings.ToLower(p))
		switch p {
		case "":
			continue
		case "html", "md":
			out = append(out, p)
		default:
			return nil, fmt.Errorf("invalid --view %q (want html, md, or html,md)", p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("--view selects no view")
	}
	return out, nil
}

// printResult writes a short human summary of one capture to stdout.
func printResult(cmd *cobra.Command, res *archive.Result) {
	if res == nil {
		return
	}
	out := cmd.OutOrStdout()
	if res.DryRun {
		fmt.Fprintf(out, "dry run · %s → %s\n", res.Target.Display, res.Root)
		return
	}
	fmt.Fprintf(out, "%s\n", res.Target.Display)
	fmt.Fprintf(out, "  repo:    %s\n", res.Root)
	fmt.Fprintf(out, "  tweets:  %d total (+%d new)", res.Total, res.Added)
	if res.Windows > 0 {
		fmt.Fprintf(out, " across %d month-windows", res.Windows)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  threads: %d\n", res.Threads)
	if res.Oldest != "" {
		fmt.Fprintf(out, "  range:   %s … %s\n", res.Oldest, res.Newest)
	}
	fmt.Fprintf(out, "  media:   %d local", res.MediaOK)
	if res.MediaFail > 0 {
		fmt.Fprintf(out, ", %d unavailable", res.MediaFail)
	}
	if res.StreamOnly > 0 {
		fmt.Fprintf(out, ", %d stream-only", res.StreamOnly)
	}
	fmt.Fprintln(out)
	if len(res.Tiers) > 0 {
		fmt.Fprintf(out, "  tiers:   %s\n", strings.Join(res.Tiers, ", "))
	}
	for _, n := range res.Notes() {
		fmt.Fprintf(out, "  note:    %s\n", n)
	}
}

// defaultOut is the output root: $TORI_OUT, else $HOME/data/tori.
func defaultOut() string {
	if v := os.Getenv("TORI_OUT"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "tori-out"
	}
	return filepath.Join(home, "data", "tori")
}

func mustString(f interface{ GetString(string) (string, error) }, name string) string {
	v, _ := f.GetString(name)
	return v
}
