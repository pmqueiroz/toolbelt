package bitwarden

import (
	"bytes"
	"strings"
	"testing"
)

func sdkVectorKey() Key {
	var key Key
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

var (
	sdkVectorIV   = []byte{216, 218, 36, 0, 196, 186, 150, 85, 49, 147, 110, 168, 185, 227, 42, 172}
	sdkVectorData = []byte{234, 77, 16, 15, 189, 82, 36, 188, 182, 88, 64, 67, 145, 94, 30, 178, 36, 235, 130, 67, 255, 207, 183, 168, 73, 231, 82, 122, 193, 139, 25, 129}
	sdkVectorMAC  = []byte{60, 78, 44, 111, 72, 233, 3, 6, 86, 250, 217, 242, 62, 229, 184, 221, 231, 150, 189, 44, 99, 189, 220, 55, 196, 194, 101, 60, 102, 195, 149, 130}
)

func TestEncryptMatchesTheSdkVector(t *testing.T) {
	enc, err := encryptWithIV(sdkVectorKey(), sdkVectorIV, []byte("Bitwarden SDK test vector"))
	if err != nil {
		t.Fatal(err)
	}
	iv, data, mac, err := parseEncString(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(iv, sdkVectorIV) || !bytes.Equal(data, sdkVectorData) || !bytes.Equal(mac, sdkVectorMAC) {
		t.Fatalf("encString %s does not match the sdk vector", enc)
	}
}

func TestDecryptRoundTrip(t *testing.T) {
	key := sdkVectorKey()
	for _, plaintext := range []string{"", "a", "exactly sixteen!", strings.Repeat("x", 100)} {
		enc, err := Encrypt(key, []byte(plaintext))
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecryptString(key, enc)
		if err != nil || got != plaintext {
			t.Errorf("round trip of %q = %q, %v", plaintext, got, err)
		}
	}
}

func TestDecryptRejectsTamperedMac(t *testing.T) {
	key := sdkVectorKey()
	enc, err := Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	other := sdkVectorKey()
	other[keySize-1] ^= 1
	if _, err := Decrypt(other, enc); err != ErrIntegrity {
		t.Fatalf("Decrypt with wrong mac key = %v; want ErrIntegrity", err)
	}
}

func TestDecryptRejectsMalformedEncStrings(t *testing.T) {
	for _, enc := range []string{"", "2.", "0.abc|def", "2.YQ==|YQ==|YQ==", "2.a|b"} {
		if _, err := Decrypt(sdkVectorKey(), enc); err == nil {
			t.Errorf("Decrypt(%q) accepted a malformed encString", enc)
		}
	}
}

func TestKeyFromBytesRequires64Bytes(t *testing.T) {
	if _, err := KeyFromBytes(make([]byte, 32)); err != ErrInvalidKey {
		t.Fatalf("KeyFromBytes(32 bytes) = %v; want ErrInvalidKey", err)
	}
	if _, err := KeyFromBytes(make([]byte, 64)); err != nil {
		t.Fatal(err)
	}
}
