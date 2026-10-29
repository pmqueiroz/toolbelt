package bitwarden

import (
	"encoding/base64"
	"testing"
)

const sdkTestToken = "0.ec2c1d46-6a4b-4751-a310-af9601317f2d.C2IgxjjLF7qSshsbwe8JGcbM075YXw:X8vbvA0bduihIDe/qrzIQQ=="

func TestParseAccessTokenDerivesTheSdkKey(t *testing.T) {
	token, err := ParseAccessToken(sdkTestToken)
	if err != nil {
		t.Fatal(err)
	}
	if token.ID != "ec2c1d46-6a4b-4751-a310-af9601317f2d" || token.ClientSecret != "C2IgxjjLF7qSshsbwe8JGcbM075YXw" {
		t.Fatalf("token = %+v", token)
	}
	want := "H9/oIRLtL9nGCQOVDjSMoEbJsjWXSOCb3qeyDt6ckzS3FhyboEDWyTP/CQfbIszNmAVg2ExFganG1FVFGXO/Jg=="
	if got := base64.StdEncoding.EncodeToString(token.Key[:]); got != want {
		t.Fatalf("key = %s; want %s", got, want)
	}
}

func TestParseAccessTokenAcceptsMissingPadding(t *testing.T) {
	if _, err := ParseAccessToken("0.ec2c1d46-6a4b-4751-a310-af9601317f2d.C2IgxjjLF7qSshsbwe8JGcbM075YXw:X8vbvA0bduihIDe/qrzIQQ"); err != nil {
		t.Fatal(err)
	}
}

func TestParseAccessTokenRejectsMalformedTokens(t *testing.T) {
	for _, raw := range []string{
		"",
		"0.ec2c1d46-6a4b-4751-a310-af9601317f2d.C2IgxjjLF7qSshsbwe8JGcbM075YXw",
		"1.ec2c1d46-6a4b-4751-a310-af9601317f2d.C2IgxjjLF7qSshsbwe8JGcbM075YXw:X8vbvA0bduihIDe/qrzIQQ==",
		"0.not-a-uuid.C2IgxjjLF7qSshsbwe8JGcbM075YXw:X8vbvA0bduihIDe/qrzIQQ==",
		"0.ec2c1d46-6a4b-4751-a310-af9601317f2d:X8vbvA0bduihIDe/qrzIQQ==",
		"0.ec2c1d46-6a4b-4751-a310-af9601317f2d.C2IgxjjLF7qSshsbwe8JGcbM075YXw:YWJj",
		"0.ec2c1d46-6a4b-4751-a310-af9601317f2d.C2IgxjjLF7qSshsbwe8JGcbM075YXw:!!!",
	} {
		if _, err := ParseAccessToken(raw); err == nil {
			t.Errorf("ParseAccessToken(%q) accepted a malformed token", raw)
		}
	}
}
