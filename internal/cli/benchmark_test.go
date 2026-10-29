package cli

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/pmqueiroz/toolbelt/internal/bench"
)

func TestParseBenchmarkDurationRejectsInvalidAndNonPositive(t *testing.T) {
	if d, err := parseBenchmarkDuration("5m"); err != nil || d != 5*time.Minute {
		t.Errorf("parseBenchmarkDuration(5m) = %s, %v", d, err)
	}
	for _, s := range []string{"", "soon", "0s", "-1m"} {
		if _, err := parseBenchmarkDuration(s); err == nil {
			t.Errorf("parseBenchmarkDuration(%q) accepted", s)
		}
	}
}

func TestRenderBenchmarkProgressShowsTimeCountAndBucket(t *testing.T) {
	out := ansi.Strip(renderBenchmarkProgress(benchmarkProgress{
		Elapsed:   2 * time.Second,
		Duration:  4 * time.Second,
		Generated: 1000,
		Set:       25,
		Indexes:   100,
	}))
	for _, want := range []string{"2s / 4s", "1,000 · 500/s", "25.0% set"} {
		if !strings.Contains(out, want) {
			t.Errorf("progress %q missing %q", out, want)
		}
	}
}

func TestRenderBenchmarkShowsEveryEstimate(t *testing.T) {
	result := bench.Result{Generated: 1 << 20, Found: 7, Elapsed: time.Second, Workers: 4}
	out := ansi.Strip(renderBenchmark(result, bench.NewBucket(2), bench.Capacity{Memory: 1 << 30, Disk: 1 << 40}))
	for _, want := range []string{"1,048,576 in 1s", "4 workers", "7 uuids", "fill memory", "1.0 GiB ram", "fill storage", "1.0 TiB free", "find every uuid", "years"} {
		if !strings.Contains(out, want) {
			t.Errorf("results missing %q:\n%s", want, out)
		}
	}
}

func TestLogFractionGivesTheLongestAFullBar(t *testing.T) {
	if got := logFraction(1e30, 1e30); got != 1 {
		t.Errorf("logFraction(longest) = %g, want 1", got)
	}
	if got := logFraction(99, 9999); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("logFraction(99, 9999) = %g, want 0.5", got)
	}
	if got := logFraction(5, math.Inf(1)); got != 1 {
		t.Errorf("logFraction with infinite longest = %g, want 1", got)
	}
}
