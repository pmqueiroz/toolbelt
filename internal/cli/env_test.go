package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestEnvRowsSortsByNameAndCompactsValues(t *testing.T) {
	rows := envRows([]string{"ZED=abcd" + strings.Repeat("x", 300) + "wxyz", "", "ALPHA=abc", "NOSEP"})
	want := [][]string{{"ALPHA", "***"}, {"NOSEP", ""}, {"ZED", "abcd******wxyz"}}
	if len(rows) != len(want) {
		t.Fatalf("envRows = %q; want %q", rows, want)
	}
	for i := range want {
		if rows[i][0] != want[i][0] || rows[i][1] != want[i][1] {
			t.Errorf("envRows[%d] = %q; want %q", i, rows[i], want[i])
		}
	}
}

func TestRenderEnvTableKeepsEveryVariableOnOneLineWithinWidth(t *testing.T) {
	long := "LS_COLORS=" + strings.Repeat("di=01;34:", 300)
	out := renderEnvTable([]string{long, "HOME=/home/someone"}, 60)
	lines := strings.Split(out, "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines (borders, header, separator, 2 rows), got %d:\n%s", len(lines), out)
	}
	for _, line := range lines {
		if w := ansi.StringWidth(line); w > 60 {
			t.Errorf("line wider than 60 (%d): %q", w, line)
		}
	}
}

func TestPrintEnvironWritesPlainLinesWhenNotATerminal(t *testing.T) {
	var buf bytes.Buffer
	printEnviron(&buf, []string{"TOKEN=abcd1234wxyz"})
	if got := buf.String(); got != "TOKEN=abcd****wxyz\n" {
		t.Errorf("printEnviron = %q", got)
	}
}

func TestRenderEnvTableStaysNarrowWhenItFits(t *testing.T) {
	out := renderEnvTable([]string{"HOME=/home/someone"}, 200)
	if w := ansi.StringWidth(strings.Split(out, "\n")[0]); w >= 60 {
		t.Errorf("expected a narrow table, got width %d:\n%s", w, out)
	}
}
