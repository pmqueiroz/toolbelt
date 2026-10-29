package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestBarFillsProportionallyAndClamps(t *testing.T) {
	cases := []struct {
		fraction float64
		filled   int
	}{
		{0, 0},
		{0.5, 5},
		{0.26, 3},
		{1, 10},
		{-1, 0},
		{2, 10},
	}
	for _, c := range cases {
		bar := ansi.Strip(Bar(10, c.fraction))
		if got := strings.Count(bar, "█"); got != c.filled {
			t.Errorf("Bar(10, %g) filled %d, want %d", c.fraction, got, c.filled)
		}
		if got := len([]rune(bar)); got != 10 {
			t.Errorf("Bar(10, %g) width %d, want 10", c.fraction, got)
		}
	}
}

func TestPanelFramesTitleAndBody(t *testing.T) {
	out := ansi.Strip(Panel("title", "body"))
	for _, want := range []string{"╭", "title", "body", "╯"} {
		if !strings.Contains(out, want) {
			t.Errorf("panel %q missing %q", out, want)
		}
	}
}
