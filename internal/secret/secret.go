package secret

import (
	"errors"
	"strings"
)

const (
	visible  = 8
	maxStars = 6
)

var ErrTooShort = errors.New("env var is too short")

func Mask(value string) (string, error) {
	runes := []rune(value)
	if len(runes) < visible {
		return "", ErrTooShort
	}
	return hideMiddle(runes, len(runes)-visible), nil
}

func Compact(value string) string {
	runes := []rune(value)
	if len(runes) < visible {
		return strings.Repeat("*", min(len(runes), maxStars))
	}
	return hideMiddle(runes, min(len(runes)-visible, maxStars))
}

func hideMiddle(runes []rune, stars int) string {
	edge := visible / 2
	return string(runes[:edge]) + strings.Repeat("*", stars) + string(runes[len(runes)-edge:])
}

func Redact(value string) string {
	if masked, err := Mask(value); err == nil {
		return masked
	}
	return strings.Repeat("*", len([]rune(value)))
}

func MaskEnviron(environ []string) []string {
	lines := make([]string, 0, len(environ))
	for _, entry := range environ {
		name, value, ok := SplitEnv(entry)
		if !ok {
			lines = append(lines, entry)
			continue
		}
		lines = append(lines, name+"="+Redact(value))
	}
	return lines
}

func SplitEnv(entry string) (name, value string, ok bool) {
	i := separator(entry)
	if i < 0 {
		return "", "", false
	}
	return entry[:i], entry[i+1:], true
}

func separator(entry string) int {
	if entry == "" {
		return -1
	}
	i := strings.Index(entry[1:], "=")
	if i < 0 {
		return -1
	}
	return i + 1
}
