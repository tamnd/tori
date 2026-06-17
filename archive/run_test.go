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
