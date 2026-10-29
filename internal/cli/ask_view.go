package cli

import (
	"strings"

	"github.com/pmqueiroz/toolbelt/internal/ai"
	"github.com/pmqueiroz/toolbelt/internal/ui"
)

const maxAnswerWidth = 100

func renderAnswer(answer ai.Answer, width int) string {
	parts := make([]string, 0, len(answer.Blocks))
	for _, b := range answer.Blocks {
		switch b.Type {
		case ai.Text:
			parts = append(parts, ui.Paragraph(b.Text, width))
		case ai.Command:
			parts = append(parts, ui.CommandLine(b.Code, b.Text, width))
		case ai.Code:
			parts = append(parts, ui.CodeBlock(b.Language, b.Code))
		case ai.List:
			parts = append(parts, ui.Bullets(b.Items, width))
		case ai.Warning:
			parts = append(parts, ui.Caution(b.Text, width))
		}
	}
	return strings.Join(parts, "\n\n")
}

func renderAnswerPlain(answer ai.Answer) string {
	var out strings.Builder
	for _, b := range answer.Blocks {
		switch b.Type {
		case ai.Text:
			out.WriteString(b.Text + "\n")
		case ai.Command, ai.Code:
			out.WriteString(b.Code + "\n")
		case ai.List:
			for _, item := range b.Items {
				out.WriteString("- " + item + "\n")
			}
		case ai.Warning:
			out.WriteString("warning: " + b.Text + "\n")
		}
	}
	return out.String()
}
