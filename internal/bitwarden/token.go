package bitwarden

import (
	"encoding/base64"
	"errors"
	"strings"
)

const (
	seedSize       = 16
	derivationName = "accesstoken"
	tokenPurpose   = "sm-access-token"
)

var ErrInvalidToken = errors.New("invalid access token, expected 0.<id>.<secret>:<key>")

type AccessToken struct {
	ID           string
	ClientSecret string
	Key          Key
}

func ParseAccessToken(raw string) (AccessToken, error) {
	credentials, encodedSeed, ok := strings.Cut(strings.TrimSpace(raw), ":")
	if !ok {
		return AccessToken{}, ErrInvalidToken
	}
	parts := strings.Split(credentials, ".")
	if len(parts) != 3 || parts[0] != "0" || !isUUID(parts[1]) || parts[2] == "" {
		return AccessToken{}, ErrInvalidToken
	}
	seed, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(encodedSeed, "="))
	if err != nil || len(seed) != seedSize {
		return AccessToken{}, ErrInvalidToken
	}
	return AccessToken{
		ID:           parts[1],
		ClientSecret: parts[2],
		Key:          deriveShareableKey(seed, derivationName, tokenPurpose),
	}, nil
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return false
			}
		}
	}
	return true
}
