package ai

import (
	"reflect"
	"testing"
)

func TestEveryPresetButOllamaNeedsAKey(t *testing.T) {
	for _, p := range Presets() {
		if p.NeedsKey() == (p.Name == "ollama") {
			t.Errorf("%s NeedsKey = %v", p.Name, p.NeedsKey())
		}
		if !p.anthropic && p.BaseURL == "" {
			t.Errorf("%s has no base url", p.Name)
		}
	}
}

func TestSortedModelsPutsThePreferredFirst(t *testing.T) {
	got := sortedModels([]string{"c", "b", "a"}, "b")
	if !reflect.DeepEqual(got, []string{"b", "a", "c"}) {
		t.Fatalf("sortedModels = %v", got)
	}
	if got := sortedModels([]string{"b", "a"}, "z"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("sortedModels = %v", got)
	}
}
