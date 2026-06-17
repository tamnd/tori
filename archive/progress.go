package archive

import (
	"fmt"

	"github.com/tamnd/x-cli/x"
)

// User aliases the engine's profile type so the Result reads naturally without a
// second import at every call site.
type User = x.User

// Logf is the optional progress sink the capture writes human-readable lines to.
// The CLI passes a function that prints to stderr; a nil Logf is silent, which is
// what the tests use so they stay quiet and deterministic.
type Logf func(format string, args ...any)

// say writes a progress line when a sink is set.
func say(log Logf, format string, args ...any) {
	if log != nil {
		log(format, args...)
	}
}

// Result summarises a completed capture for the CLI to print and for the exit
// code to reflect (a partial capture with failures is still a success on disk).
type Result struct {
	Root       string   // repository directory written
	Target     Target   // what was captured
	Profile    *User    // the captured profile, when the target had one
	Added      int      // tweets newly written this run
	Total      int      // tweets in the repo after this run
	Threads    int      // reconstructed conversations
	MediaOK    int      // media files on disk after this run
	MediaFail  int      // media references that could not be localised
	StreamOnly int      // video with no progressive rendition
	Tiers      []string // tiers that served this capture
	Oldest     string   // oldest captured tweet timestamp (RFC3339), empty if none
	Newest     string   // newest captured tweet timestamp
	DryRun     bool     // true when nothing was fetched or written
	Windows    int      // month-windows walked for a full-history profile capture
	notes      []string // human notes (degradations, hints)
}

// Note records a one-line degradation or hint for the CLI to surface (e.g. a
// Tier-0 capture that --guest would page deeper).
func (r *Result) Note(format string, args ...any) {
	r.notes = append(r.notes, fmt.Sprintf(format, args...))
}

// Notes returns the accumulated notes.
func (r *Result) Notes() []string { return r.notes }
