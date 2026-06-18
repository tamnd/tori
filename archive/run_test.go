package archive

import (
	"testing"
	"time"

	"github.com/tamnd/x-cli/x"
)

func TestMonthWindowsNewestFirst(t *testing.T) {
	from := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	w := monthWindows(from, to)
	// Windows cover Jan, Feb, and the partial March, newest-first.
	if len(w) != 3 {
		t.Fatalf("want 3 windows, got %d: %v", len(w), w)
	}
	// Each window is a [start,end) of exactly one month, aligned to the 1st.
	for i, span := range w {
		if span[0].Day() != 1 || span[1].Day() != 1 {
			t.Errorf("window %d not month-aligned: %v", i, span)
		}
		if !span[0].Before(span[1]) {
			t.Errorf("window %d not ordered: %v", i, span)
		}
	}
	// Newest first: the first window ends after the last window starts.
	if !w[0][0].After(w[len(w)-1][0]) {
		t.Errorf("windows not newest-first: %v", w)
	}
	// The newest window includes March (the current, partial month).
	if w[0][1].Month() != time.April {
		t.Errorf("newest window should end at the month boundary after March, got %v", w[0][1])
	}
}

func TestMonthWindowsContiguous(t *testing.T) {
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	w := monthWindows(from, to)
	// Each window's start must equal the previous window's start minus a month:
	// no gaps, no overlaps.
	for i := 1; i < len(w); i++ {
		if !w[i][1].Equal(w[i-1][0]) {
			t.Errorf("gap between window %d and %d: %v / %v", i-1, i, w[i-1], w[i])
		}
	}
}

// TestHybridWindowClamp covers the arithmetic at the heart of the hybrid capture:
// pass 1 streams the timeline back to some oldest reach, and pass 2 searches only
// the windows older than that. So clamping the search end to the timeline's oldest
// date must drop the windows the timeline already covered, and leave none at all
// when the timeline reached the account's creation (an account under ~3200 posts).
func TestHybridWindowClamp(t *testing.T) {
	from := time.Date(2009, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	full := monthWindows(from, now)

	// Timeline reached back to 2023-08: search covers only 2009-06 .. 2023-08.
	reached := time.Date(2023, 8, 12, 0, 0, 0, 0, time.UTC)
	clamped := monthWindows(from, reached)
	if len(clamped) >= len(full) {
		t.Fatalf("clamp did not reduce windows: %d full vs %d clamped", len(full), len(clamped))
	}
	// The newest clamped window must not reach past the timeline's oldest month.
	if clamped[0][1].After(time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("clamped search overlaps the timeline beyond its boundary month: %v", clamped[0])
	}

	// Timeline reached the account's creation month: at most the one boundary month
	// is left to double-check, not the whole back catalogue.
	if got := monthWindows(from, from); len(got) > 1 {
		t.Errorf("timeline back to creation should leave <=1 search window, got %d", len(got))
	}
	// Timeline reached before the creation month: search has nothing left to do.
	if got := monthWindows(from, from.AddDate(0, -1, 0)); len(got) != 0 {
		t.Errorf("timeline past creation should leave 0 search windows, got %d", len(got))
	}
}

func TestIDNewer(t *testing.T) {
	if !idNewer("100", "99") {
		t.Error("longer id should be newer")
	}
	if !idNewer("205", "200") {
		t.Error("205 should be newer than 200")
	}
	if idNewer("200", "200") {
		t.Error("equal ids are not strictly newer")
	}
}

func TestNewestID(t *testing.T) {
	if got := newestID(nil); got != "" {
		t.Errorf("empty set newest = %q, want empty", got)
	}
	set := []*x.Tweet{{ID: "200"}, {ID: "205"}, {ID: "100"}}
	if got := newestID(set); got != "205" {
		t.Errorf("newest = %q, want 205", got)
	}
}
