package merge

import "testing"

func merged(t *testing.T, base, local, upstream string) (string, int) {
	t.Helper()
	r := Three([]byte(base), []byte(local), []byte(upstream))
	return string(r.Content), r.Conflicts
}

func TestThreeKeepsLocalEditsAndTakesUpstreamEdits(t *testing.T) {
	base := "a\nb\nc\nd\ne\n"
	local := "a\nB local\nc\nd\ne\n"
	upstream := "a\nb\nc\nd\nE upstream\n"

	got, conflicts := merged(t, base, local, upstream)
	if conflicts != 0 || got != "a\nB local\nc\nd\nE upstream\n" {
		t.Errorf("merge = %q, conflicts = %d", got, conflicts)
	}
}

func TestThreeTakesUpstreamWhenLocalIsUntouched(t *testing.T) {
	got, conflicts := merged(t, "a\nb\n", "a\nb\n", "a\nb\nc\n")
	if conflicts != 0 || got != "a\nb\nc\n" {
		t.Errorf("merge = %q, conflicts = %d", got, conflicts)
	}
}

func TestThreeKeepsLocalWhenUpstreamIsUntouched(t *testing.T) {
	got, conflicts := merged(t, "a\nb\n", "x\na\nb\n", "a\nb\n")
	if conflicts != 0 || got != "x\na\nb\n" {
		t.Errorf("merge = %q, conflicts = %d", got, conflicts)
	}
}

func TestThreeAcceptsTheSameChangeOnBothSides(t *testing.T) {
	got, conflicts := merged(t, "a\nb\n", "a\nc\n", "a\nc\n")
	if conflicts != 0 || got != "a\nc\n" {
		t.Errorf("merge = %q, conflicts = %d", got, conflicts)
	}
}

func TestThreeKeepsBothSidesOfAConflict(t *testing.T) {
	got, conflicts := merged(t, "a\nb\nc\n", "a\nlocal\nc\n", "a\nupstream\nc\n")
	want := "a\n<<<<<<< local\nlocal\n=======\nupstream\n>>>>>>> toolbelt\nc\n"
	if conflicts != 1 || got != want {
		t.Errorf("merge = %q, conflicts = %d", got, conflicts)
	}
	if !HasMarkers([]byte(got)) {
		t.Error("HasMarkers = false on a conflicted merge")
	}
}

func TestThreeTerminatesConflictLinesWithoutTrailingNewline(t *testing.T) {
	got, conflicts := merged(t, "a\nb", "a\nlocal", "a\nupstream")
	want := "a\n<<<<<<< local\nlocal\n=======\nupstream\n>>>>>>> toolbelt\n"
	if conflicts != 1 || got != want {
		t.Errorf("merge = %q, conflicts = %d", got, conflicts)
	}
}

func TestThreeHandlesEmptyBase(t *testing.T) {
	got, conflicts := merged(t, "", "a\n", "a\n")
	if conflicts != 0 || got != "a\n" {
		t.Errorf("merge = %q, conflicts = %d", got, conflicts)
	}
}

func TestHasMarkersIgnoresLookalikes(t *testing.T) {
	if HasMarkers([]byte("a\n<<<<<<< other\n=======\n")) {
		t.Error("HasMarkers matched a line tb never writes")
	}
}
