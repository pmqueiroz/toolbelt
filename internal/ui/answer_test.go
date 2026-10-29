package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestCommandLinePrefixesEveryLineWithAPrompt(t *testing.T) {
	out := ansi.Strip(CommandLine("cd /tmp\nls", "list temp", 40))
	for _, want := range []string{"$ cd /tmp", "$ ls", "list temp"} {
		if !strings.Contains(out, want) {
			t.Errorf("command line %q missing %q", out, want)
		}
	}
}

func TestCodeBlockShowsTheLanguageAboveTheCode(t *testing.T) {
	out := ansi.Strip(CodeBlock("go", "fmt.Println()"))
	if strings.Index(out, "go") > strings.Index(out, "fmt.Println()") {
		t.Errorf("code block %q: language should come first", out)
	}
}

func TestBulletsRenderOneLinePerItem(t *testing.T) {
	out := ansi.Strip(Bullets([]string{"one", "two"}, 40))
	if strings.Count(out, "•") != 2 {
		t.Errorf("bullets %q, want 2", out)
	}
}

func TestParagraphWrapsToWidth(t *testing.T) {
	out := ansi.Strip(Paragraph("the moon is the natural satellite of the earth", 10))
	for _, line := range strings.Split(out, "\n") {
		if len(strings.TrimRight(line, " ")) > 10 {
			t.Errorf("line %q wider than 10", line)
		}
	}
}
