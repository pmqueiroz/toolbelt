package cli

import "testing"

func TestMask(t *testing.T) {
	cases := map[string]string{
		"abcdefgh":     "abcdefgh",
		"abcd1234wxyz": "abcd****wxyz",
	}
	for in, want := range cases {
		got, err := mask(in)
		if err != nil || got != want {
			t.Errorf("mask(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := mask("short"); err == nil {
		t.Error("expected error for short value")
	}
}
