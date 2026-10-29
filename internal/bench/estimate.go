package bench

import (
	"fmt"
	"math"
	"time"
)

const (
	UUIDBytes          = 16
	randomBits         = 122
	eulerGamma         = 0.5772156649015329
	secondsPerYear     = 365.25 * 24 * 60 * 60
	yearsPerMillennium = 1000
	EarthStorage       = 18e21
)

var UUIDSpace = math.Ldexp(1, randomBits)

var largeNumbers = []string{"", "thousand", "million", "billion", "trillion", "quadrillion", "quintillion", "sextillion", "septillion", "octillion", "nonillion", "decillion"}

func SecondsToFindAll(rate float64) float64 {
	draws := UUIDSpace * (math.Log(UUIDSpace) + eulerGamma)
	return secondsFor(draws, rate)
}

func SecondsToFill(capacity, rate float64) float64 {
	return secondsFor(capacity/UUIDBytes, rate)
}

func secondsFor(count, rate float64) float64 {
	if rate <= 0 {
		return math.Inf(1)
	}
	return count / rate
}

func Years(seconds float64) float64 {
	return seconds / secondsPerYear
}

func FormatSeconds(seconds float64) string {
	switch {
	case math.IsInf(seconds, 1):
		return "never"
	case seconds >= secondsPerYear*yearsPerMillennium:
		return fmt.Sprintf("%.3g years (%s)", Years(seconds), DescribeYears(Years(seconds)))
	case seconds >= secondsPerYear:
		return fmt.Sprintf("%.3g years", Years(seconds))
	case seconds >= 1:
		return time.Duration(seconds * float64(time.Second)).Round(time.Second).String()
	default:
		return time.Duration(seconds * float64(time.Second)).Round(time.Microsecond).String()
	}
}

func DescribeYears(years float64) string {
	unit, count := "years", years
	if years >= yearsPerMillennium {
		unit, count = "millennia", years/yearsPerMillennium
	}
	group := 0
	for count >= 999.5 && group < len(largeNumbers)-1 {
		count /= 1000
		group++
	}
	if math.Round(count) == 1 && group == 0 {
		unit = map[string]string{"years": "year", "millennia": "millennium"}[unit]
	}
	if group == 0 {
		return fmt.Sprintf("%.0f %s", count, unit)
	}
	return fmt.Sprintf("%.0f %s %s", count, largeNumbers[group], unit)
}
