package bitwarden

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const (
	keySize      = 64
	halfKeySize  = 32
	encTypeCBC   = "2"
	encSeparator = "|"
)

var (
	ErrInvalidKey       = errors.New("invalid key, expected 64 bytes")
	ErrInvalidEncString = errors.New("invalid encrypted string")
	ErrIntegrity        = errors.New("encrypted string failed the integrity check")
)

type Key [keySize]byte

func (k Key) enc() []byte { return k[:halfKeySize] }
func (k Key) mac() []byte { return k[halfKeySize:] }

func KeyFromBytes(raw []byte) (Key, error) {
	var key Key
	if len(raw) != keySize {
		return key, ErrInvalidKey
	}
	copy(key[:], raw)
	return key, nil
}

func deriveShareableKey(seed []byte, name, info string) Key {
	prk := hmac.New(sha256.New, []byte("bitwarden-"+name))
	prk.Write(seed)
	var key Key
	io.ReadFull(hkdf.Expand(sha256.New, prk.Sum(nil), []byte(info)), key[:])
	return key
}

func Encrypt(key Key, plaintext []byte) (string, error) {
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	return encryptWithIV(key, iv, plaintext)
}

func encryptWithIV(key Key, iv, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key.enc())
	if err != nil {
		return "", err
	}
	data := pad(plaintext)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(data, data)
	return encTypeCBC + "." + strings.Join([]string{
		base64.StdEncoding.EncodeToString(iv),
		base64.StdEncoding.EncodeToString(data),
		base64.StdEncoding.EncodeToString(sign(key, iv, data)),
	}, encSeparator), nil
}

func Decrypt(key Key, encString string) ([]byte, error) {
	iv, data, mac, err := parseEncString(encString)
	if err != nil {
		return nil, err
	}
	if !hmac.Equal(mac, sign(key, iv, data)) {
		return nil, ErrIntegrity
	}
	block, err := aes.NewCipher(key.enc())
	if err != nil {
		return nil, err
	}
	plaintext := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, data)
	return unpad(plaintext)
}

func DecryptString(key Key, encString string) (string, error) {
	plaintext, err := Decrypt(key, encString)
	return string(plaintext), err
}

func parseEncString(encString string) (iv, data, mac []byte, err error) {
	encType, body, ok := strings.Cut(encString, ".")
	if !ok || encType != encTypeCBC {
		return nil, nil, nil, ErrInvalidEncString
	}
	parts := strings.Split(body, encSeparator)
	if len(parts) != 3 {
		return nil, nil, nil, ErrInvalidEncString
	}
	decoded := make([][]byte, len(parts))
	for i, part := range parts {
		if decoded[i], err = base64.StdEncoding.DecodeString(part); err != nil {
			return nil, nil, nil, ErrInvalidEncString
		}
	}
	iv, data, mac = decoded[0], decoded[1], decoded[2]
	if len(iv) != aes.BlockSize || len(data) == 0 || len(data)%aes.BlockSize != 0 || len(mac) != sha256.Size {
		return nil, nil, nil, ErrInvalidEncString
	}
	return iv, data, mac, nil
}

func sign(key Key, iv, data []byte) []byte {
	mac := hmac.New(sha256.New, key.mac())
	mac.Write(iv)
	mac.Write(data)
	return mac.Sum(nil)
}

func pad(plaintext []byte) []byte {
	n := aes.BlockSize - len(plaintext)%aes.BlockSize
	return append(bytes.Clone(plaintext), bytes.Repeat([]byte{byte(n)}, n)...)
}

func unpad(data []byte) ([]byte, error) {
	n := int(data[len(data)-1])
	if n == 0 || n > aes.BlockSize || n > len(data) {
		return nil, ErrInvalidEncString
	}
	for _, b := range data[len(data)-n:] {
		if int(b) != n {
			return nil, ErrInvalidEncString
		}
	}
	return data[:len(data)-n], nil
}
