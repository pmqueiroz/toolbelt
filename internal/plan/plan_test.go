package plan

import "testing"

func TestDedupeKeepsFirstOccurrenceInOrder(t *testing.T) {
	actions := []Action{
		{Description: "install mise"},
		{Description: "mise use -g node@lts"},
		{Description: "install mise"},
		{Description: "mise use -g pnpm@latest"},
	}

	got := Dedupe(actions)

	want := []string{"install mise", "mise use -g node@lts", "mise use -g pnpm@latest"}
	if len(got) != len(want) {
		t.Fatalf("got %d actions, want %d", len(got), len(want))
	}
	for i, a := range got {
		if a.Description != want[i] {
			t.Errorf("action %d = %q, want %q", i, a.Description, want[i])
		}
	}
}
