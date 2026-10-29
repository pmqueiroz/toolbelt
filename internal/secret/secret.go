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
