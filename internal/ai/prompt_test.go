package ai

import (
	"strings"
	"testing"
)

func TestSystemNamesTheUsersOSAndShell(t *testing.T) {
	got := System(Question{OS: "linux", Shell: "zsh"})
	if !strings.Contains(got, "User OS: linux. Shell: zsh.") {
		t.Fatalf("system prompt missing environment: %q", got)
	}
	if !strings.Contains(System(Question{OS: "darwin"}), "Shell: unknown.") {
		t.Fatal("empty shell should read unknown")
	}
}

func TestUserIsTheQuestionAloneWithoutContext(t *testing.T) {
	if got := User(Question{Text: " what is the moon? "}); got != "what is the moon?" {
		t.Fatalf("User = %q", got)
	}
}

func TestUserAppendsPipedContext(t *testing.T) {
	got := User(Question{Text: "why", Context: "panic: boom\n"})
	if !strings.HasPrefix(got, "why\n\n") || !strings.HasSuffix(got, "panic: boom") {
		t.Fatalf("User = %q", got)
	}
}

func TestUserKeepsTheEndOfLongContext(t *testing.T) {
	context := strings.Repeat("a", MaxContextBytes) + "TAIL"
	got := User(Question{Text: "why", Context: context})
	_, sent, _ := strings.Cut(got, ":\n")
	if len(sent) != MaxContextBytes || !strings.HasSuffix(sent, "TAIL") {
		t.Fatalf("context not trimmed to its last %d bytes", MaxContextBytes)
	}
}
