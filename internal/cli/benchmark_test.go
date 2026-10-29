package cli

import (
	"testing"
	"time"
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
