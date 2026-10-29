package auth

import (
	"strings"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/dots"
)

func TestLastSecretKeyID(t *testing.T) {
	colons := `sec:u:4096:1:AAAA1111BBBB2222:1700000000:::u:::scESC:::+:::23::0:
fpr:::::::::0123456789ABCDEF0123AAAA1111BBBB2222:
uid:u::::1700000000::HASH::peam <peam@example.com>::::::::::0:
ssb:u:4096:1:CCCC3333DDDD4444:1700000000::::::e:::+:::23:
sec:u:4096:1:EEEE5555FFFF6666:1710000000:::u:::scESC:::+:::23::0:
`
	if got := LastSecretKeyID(colons); got != "EEEE5555FFFF6666" {
		t.Errorf("LastSecretKeyID = %q", got)
	}
	if got := LastSecretKeyID(""); got != "" {
		t.Errorf("empty listing gave %q", got)
	}
}

func TestGPGTemplateRendersEveryPlaceholder(t *testing.T) {
	got := dots.Render(gpgTemplate, dots.Vars{Username: "peam", Email: "peam@example.com", Passphrase: "s3cret"})

	if strings.Contains(got, "{{") {
		t.Errorf("unrendered placeholder in %q", got)
	}
	for _, want := range []string{"Name-Real: peam", "Name-Email: peam@example.com", "Passphrase: s3cret"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
}
