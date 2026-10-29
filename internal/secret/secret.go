package secret

import (
	"errors"
	"strings"
)

const visible = 8

var ErrTooShort = errors.New("env var is too short")

func Mask(value string) (string, error) {
	runes := []rune(value)
	if len(runes) < visible {
		return "", ErrTooShort
	}
	edge := visible / 2
	return string(runes[:edge]) + strings.Repeat("*", len(runes)-visible) + string(runes[len(runes)-edge:]), nil
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
		i := separator(entry)
		if i < 0 {
			lines = append(lines, entry)
			continue
		}
		lines = append(lines, entry[:i+1]+Redact(entry[i+1:]))
	}
	return lines
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
