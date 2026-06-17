// Package cli is the tori command tree: a cobra hierarchy wrapped with
// charmbracelet/fang for polished help, version, and error rendering (spec §12).
// The commands are thin — they parse flags, build an x-cli engine, call the
// archive/render packages, and map the typed errors those return onto stable
// exit codes (spec §6). No scraping or rendering logic lives here.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/tamnd/x-cli/x"
)

// Exit codes (spec §6). The CLI maps the engine's typed errors onto these so a
// script can branch on the outcome.
const (
	CodeOK        = 0
	CodeUsage     = 1
	CodePartial   = 2
	CodeNeedsAuth = 4
	CodeBlocked   = 5
	CodeNotFound  = 6
	CodeInterrupt = 130
)

// partialError marks a capture that finished and wrote a repository but could
// not localise every reference (exit code 2). It is informational, not fatal.
type partialError struct{ msg string }

func (e *partialError) Error() string { return e.msg }

// Execute builds the command tree, runs it through fang, and returns a process
// exit code. cmd/tori/main.go passes a signal-aware context so Ctrl-C cancels
// the in-flight capture and exits 130.
func Execute(ctx context.Context) int {
	root := newRootCmd()
	err := fang.Execute(ctx, root,
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
	)
	return codeFor(ctx, err)
}

// codeFor maps an error (already rendered by fang) onto an exit code.
func codeFor(ctx context.Context, err error) int {
	if err == nil {
		return CodeOK
	}
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return CodeInterrupt
	}
	var na *x.NeedAuthError
	if errors.As(err, &na) {
		return CodeNeedsAuth
	}
	var rl *x.RateLimitedError
	if errors.As(err, &rl) {
		return CodeBlocked
	}
	var nf *x.NotFoundError
	if errors.As(err, &nf) {
		return CodeNotFound
	}
	var he *x.HTTPError
	if errors.As(err, &he) {
		if he.Status == 429 {
			return CodeBlocked
		}
		if he.Status == 404 {
			return CodeNotFound
		}
	}
	var pe *partialError
	if errors.As(err, &pe) {
		return CodePartial
	}
	return CodeUsage
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tori",
		Short: "Build offline, browsable archives of X (Twitter) content",
		Long: "tori (鳥, bird) captures X profiles, threads, searches and more into\n" +
			"self-contained archives: canonical JSON, localised media, and inert\n" +
			"HTML and Markdown views that open with the network unplugged.\n\n" +
			"It reads X through the free tiers of the x-cli engine — no API key.\n" +
			"Tier 0 (syndication) needs no setup; --guest opens the free guest tier\n" +
			"for deeper paging; `tori auth import` uses your own session for the rest.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Access and politeness, shared by every fetching command (delegated to the
	// engine config). Output and capture flags live on the individual commands.
	pf := root.PersistentFlags()
	pf.Bool("guest", false, "use the free guest-token tier for deeper paging")
	pf.String("tier", "", "force a tier: syndication|guest|session")
	pf.Duration("rate", 0, "minimum delay between requests (default: engine default)")
	pf.Int("retries", -1, "retry attempts on a transient failure (default: engine default)")
	pf.Duration("timeout", 0, "per-request timeout (default: engine default)")
	pf.Bool("no-cache", false, "bypass the on-disk response cache")
	pf.BoolP("verbose", "v", false, "log the tier and each record as it is captured")

	root.AddCommand(
		newArchiveCmd(false),
		newArchiveCmd(true),
		newRenderCmd(),
		newServeCmd(),
		newInfoCmd(),
		newAuthCmd(),
	)
	return root
}

// engineFromFlags builds an x-cli engine from defaults, the environment, and the
// shared persistent flags. The environment supplies any imported session (TP:
// the only secrets are the user's own cookies, never an API key).
func engineFromFlags(cmd *cobra.Command) *x.Engine {
	cfg := x.DefaultConfig()
	cfg.FromEnv()

	f := cmd.Flags()
	if v, _ := f.GetBool("guest"); v {
		cfg.AllowGuest = true
	}
	if v, _ := f.GetString("tier"); v != "" {
		cfg.Tier = v
	}
	if v, _ := f.GetDuration("rate"); v > 0 {
		cfg.Rate = v
	}
	if v, _ := f.GetInt("retries"); v >= 0 {
		cfg.Retries = v
	}
	if v, _ := f.GetDuration("timeout"); v > 0 {
		cfg.Timeout = v
	}
	if v, _ := f.GetBool("no-cache"); v {
		cfg.NoCache = true
	}
	return x.NewEngine(cfg)
}

// stderrLog returns a progress sink that writes to stderr when verbose is set,
// else a nil (silent) sink.
func stderrLog(cmd *cobra.Command) func(string, ...any) {
	if v, _ := cmd.Flags().GetBool("verbose"); !v {
		return nil
	}
	return func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// parseDate parses an optional RFC3339 capture stamp.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// parseLooseTime parses a timeline bound as either RFC3339 or a bare calendar
// date (2006-01-02, interpreted as UTC midnight).
func parseLooseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
