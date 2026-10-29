package cli

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/x/term"
	"github.com/pmqueiroz/toolbelt/internal/secret"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env [value]",
		Short: "Print a secret masked, or every environment variable masked when no value is given",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(args) == 0 {
				printEnviron(out, os.Environ())
				return nil
			}
			masked, err := secret.Mask(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(out, masked)
			return nil
		},
	}
}

func printEnviron(out io.Writer, environ []string) {
	width, ok := terminalWidth(out)
	if !ok {
		for _, line := range secret.MaskEnviron(environ) {
			fmt.Fprintln(out, line)
		}
		return
	}
	fmt.Fprintln(out, renderEnvTable(environ, width))
}

func terminalWidth(out io.Writer) (int, bool) {
	f, ok := out.(*os.File)
	if !ok || !term.IsTerminal(f.Fd()) {
		return 0, false
	}
	width, _, err := term.GetSize(f.Fd())
	if err != nil || width <= 0 {
		return 0, false
	}
	return width, true
}

func renderEnvTable(environ []string, width int) string {
	rows := envRows(environ)
	nameStyle := lipgloss.NewStyle().Foreground(ui.Purple).Padding(0, 1)
	valueStyle := lipgloss.NewStyle().Padding(0, 1)
	headerStyle := ui.Title.Padding(0, 1)
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ui.Rose)).
		Headers("NAME", "VALUE").
		Rows(rows...).
		Wrap(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case col == 0:
				return nameStyle
			default:
				return valueStyle
			}
		})
	if lipgloss.Width(t.Render()) > width {
		t.Width(width)
	}
	return t.Render()
}

func envRows(environ []string) [][]string {
	rows := make([][]string, 0, len(environ))
	for _, entry := range environ {
		name, value, ok := secret.SplitEnv(entry)
		if !ok {
			if entry != "" {
				rows = append(rows, []string{entry, ""})
			}
			continue
		}
		rows = append(rows, []string{name, secret.Compact(value)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	return rows
}
