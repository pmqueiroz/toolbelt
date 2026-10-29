package setup

import (
	"slices"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/recipes"
)

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b", " peam@example.com "} {
		if err := validEmail(ok); err != nil {
			t.Errorf("validEmail(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "peam", "@example.com", "peam@"} {
		if err := validEmail(bad); err == nil {
			t.Errorf("validEmail(%q) should fail", bad)
		}
	}
}

func TestDefaultRecipes(t *testing.T) {
	got := defaultRecipes(recipes.ForOS("darwin"))
	if !slices.Equal(got, []string{"essentials", "code", "claude"}) {
		t.Errorf("defaultRecipes = %v", got)
	}
}

func TestDefaultLanguagesIsNode(t *testing.T) {
	got := defaultRecipes(recipes.LanguagesForOS("linux"))
	if !slices.Equal(got, []string{"node"}) {
		t.Errorf("defaultRecipes(languages) = %v", got)
	}
}
