package secret

import "testing"

func TestMask(t *testing.T) {
	cases := map[string]string{
		"abcdefgh":     "abcdefgh",
		"abcd1234wxyz": "abcd****wxyz",
	}
	for in, want := range cases {
		got, err := Mask(in)
		if err != nil || got != want {
			t.Errorf("Mask(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := Mask("short"); err == nil {
		t.Error("expected error for short value")
	}
}

func TestRedact(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"short":        "*****",
		"abcd1234wxyz": "abcd****wxyz",
	}
	for in, want := range cases {
		if got := Redact(in); got != want {
			t.Errorf("Redact(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestMaskEnviron(t *testing.T) {
	got := MaskEnviron([]string{"TOKEN=abcd1234wxyz", "SHORT=abc", "EMPTY=", "=C:=C:\\x", "", "NOSEP"})
	want := []string{"TOKEN=abcd****wxyz", "SHORT=***", "EMPTY=", "=C:=****", "", "NOSEP"}
	if len(got) != len(want) {
		t.Fatalf("MaskEnviron = %q; want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("MaskEnviron[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}
