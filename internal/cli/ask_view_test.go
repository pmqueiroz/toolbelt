package cli

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/pmqueiroz/toolbelt/internal/ai"
)

var sampleAnswer = ai.Answer{Blocks: []ai.Block{
	{Type: ai.Text, Text: "Deletes a branch."},
	{Type: ai.Command, Text: "local only", Code: "git branch -D feat"},
	{Type: ai.Code, Language: "go", Code: "fmt.Println()"},
	{Type: ai.List, Items: []string{"one", "two"}},
	{Type: ai.Warning, Text: "cannot be undone"},
}}

func TestRenderAnswerShowsEveryBlock(t *testing.T) {
	out := ansi.Strip(renderAnswer(sampleAnswer, 60))
	for _, want := range []string{"Deletes a branch.", "$ git branch -D feat", "local only", "go", "fmt.Println()", "• one", "• two", "⚠ cannot be undone"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderAnswerPlainIsPipeFriendly(t *testing.T) {
	want := "Deletes a branch.\ngit branch -D feat\nfmt.Println()\n- one\n- two\nwarning: cannot be undone\n"
	if got := renderAnswerPlain(sampleAnswer); got != want {
		t.Fatalf("plain = %q, want %q", got, want)
	}
}
