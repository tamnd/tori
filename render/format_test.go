package render

import "testing"

func TestFormatCount(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1K"},
		{1200, "1.2K"},
		{21500, "21.5K"},
		{999999, "1000K"},
		{1000000, "1M"},
		{2700000, "2.7M"},
	}
	for _, c := range cases {
		if got := FormatCount(c.n); got != c.want {
			t.Errorf("FormatCount(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
