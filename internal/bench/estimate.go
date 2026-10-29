package bench

import (
	"fmt"
	"math"
	"time"
)

const (
	UUIDBytes      = 16
	randomBits     = 122
	eulerGamma     = 0.5772156649015329
	secondsPerYear = 365.25 * 24 * 60 * 60
)

var UUIDSpace = math.Ldexp(1, randomBits)

func SecondsToFindAll(rate float64) float64 {
	draws := UUIDSpace * (math.Log(UUIDSpace) + eulerGamma)
	return secondsFor(draws, rate)
}

func SecondsToFill(capacity uint64, rate float64) float64 {
	return secondsFor(float64(capacity)/UUIDBytes, rate)
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
	case seconds >= secondsPerYear:
		return fmt.Sprintf("%.3g years", Years(seconds))
	case seconds >= 1:
		return time.Duration(seconds * float64(time.Second)).Round(time.Second).String()
	default:
		return time.Duration(seconds * float64(time.Second)).Round(time.Microsecond).String()
	}
}
