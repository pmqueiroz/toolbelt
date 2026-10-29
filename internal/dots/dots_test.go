package dots

import (
	"testing"
)

func TestRender(t *testing.T) {
	got := Render("name = {{username}}\nemail = {{email}}\npass = {{passphrase}}", Vars{"peam", "p@e.com", "s3cret"})
	if got != "name = peam\nemail = p@e.com\npass = s3cret" {
		t.Errorf("Render = %q", got)
	}
}

func TestEveryDotIsEmbedded(t *testing.T) {
	for _, d := range All {
		if _, err := embedded.ReadFile("files/" + d.File); err != nil {
			t.Errorf("%s: %v", d.File, err)
		}
	}
}
