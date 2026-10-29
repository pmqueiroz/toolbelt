package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

func newHotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hot <command> <file>...",
		Short: "Rerun a command every time one of the files changes",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			r := quietRunner()
			script, files := args[0], args[1:]
			shell, flag := shellCommand()

			last := ""
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				current := fingerprint(files)
				if current != last {
					last = current
					if err := r.Attached(ctx, shell, flag, script); err != nil && ctx.Err() == nil {
						ui.Error(err.Error())
					}
				}
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
				}
			}
		},
	}
}

func shellCommand() (string, string) {
	if runtime.GOOS == "windows" {
		return "powershell", "-Command"
	}
	return "sh", "-c"
}

func fingerprint(files []string) string {
	var b strings.Builder
	for _, f := range files {
		info, err := os.Stat(f)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(&b, "%s:missing;", f)
			continue
		}
		if err != nil {
			fmt.Fprintf(&b, "%s:%v;", f, err)
			continue
		}
		fmt.Fprintf(&b, "%s:%d:%d;", f, info.Size(), info.ModTime().UnixNano())
	}
	return b.String()
}
