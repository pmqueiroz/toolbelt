package bench

import (
	"math"
	"testing"
)

func TestSecondsToFindAllUsesTheCouponCollectorBound(t *testing.T) {
	want := UUIDSpace * (math.Log(UUIDSpace) + eulerGamma) / 1e6
	if got := SecondsToFindAll(1e6); got != want {
		t.Errorf("SecondsToFindAll(1e6) = %g, want %g", got, want)
	}
	if !math.IsInf(SecondsToFindAll(0), 1) {
		t.Error("zero rate should never finish")
	}
}

func TestSecondsToFillStoresSixteenBytesPerUUID(t *testing.T) {
	if got := SecondsToFill(1600, 10); got != 10 {
		t.Errorf("SecondsToFill(1600, 10) = %g, want 10", got)
	}
}

func TestFormatSecondsSwitchesToYears(t *testing.T) {
	cases := map[float64]string{
		math.Inf(1):           "never",
		0.0015:                "1.5ms",
		90.4:                  "1m30s",
		2 * secondsPerYear:    "2 years",
		3e30 * secondsPerYear: "3e+30 years (3 octillion millennia)",
	}
	for seconds, want := range cases {
		if got := FormatSeconds(seconds); got != want {
			t.Errorf("FormatSeconds(%g) = %q, want %q", seconds, got, want)
		}
	}
}

func TestDescribeYearsNamesTheMillennia(t *testing.T) {
	cases := map[float64]string{
		1:        "1 year",
		12:       "12 years",
		1000:     "1 millennium",
		1500:     "2 millennia",
		2.92e23:  "292 quintillion millennia",
		6e18:     "6 quadrillion millennia",
		9.9995e8: "1 million millennia",
		1e40:     "10000 decillion millennia",
	}
	for years, want := range cases {
		if got := DescribeYears(years); got != want {
			t.Errorf("DescribeYears(%g) = %q, want %q", years, got, want)
		}
	}
}

func TestEarthStorageTakesLongerThanAnyDisk(t *testing.T) {
	if SecondsToFill(EarthStorage, 1e6) <= SecondsToFill(1e15, 1e6) {
		t.Error("earth storage should take longer than a petabyte")
	}
}
