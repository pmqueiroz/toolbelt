package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/dustin/go-humanize"
	"github.com/pmqueiroz/toolbelt/internal/bench"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

const (
	customDuration   = "custom"
	progressInterval = 15 * time.Second
	shortBenchmark   = 5 * time.Second
	longBenchmark    = 10 * time.Second
)

func newBenchmarkCmd() *cobra.Command {
	var duration time.Duration
	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Generate random uuids for a while and estimate how long finding all of them would take",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("duration") && ui.Interactive() {
				chosen, err := askDuration()
				if err != nil {
					return err
				}
				duration = chosen
			}
			if duration <= 0 {
				return errors.New("duration must be positive")
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			capacity, err := bench.ReadCapacity(home)
			if err != nil {
				return fmt.Errorf("read memory and disk size: %w", err)
			}

			run := &bench.Run{
				Duration: duration,
				Workers:  runtime.NumCPU(),
				Bucket:   bench.NewBucket(bench.IndexChars),
			}
			result := runBenchmark(cmd.Context(), run)
			fmt.Fprintln(cmd.OutOrStdout(), renderBenchmark(result, run.Bucket, capacity))
			return nil
		},
	}
	cmd.Flags().DurationVarP(&duration, "duration", "d", shortBenchmark, "how long to generate uuids (5s, 10s or any duration like 1m)")
	return cmd
}

func askDuration() (time.Duration, error) {
	choice := shortBenchmark.String()
	custom := ""
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How long should the benchmark run?").
				Options(
					huh.NewOption("5 seconds", shortBenchmark.String()),
					huh.NewOption("10 seconds", longBenchmark.String()),
					huh.NewOption("custom", customDuration),
				).
				Value(&choice),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Duration").
				Placeholder("30s, 1m, 5m").
				Validate(func(s string) error {
					_, err := parseBenchmarkDuration(s)
					return err
				}).
				Value(&custom),
		).WithHideFunc(func() bool { return choice != customDuration }),
	).Run()
	if err != nil {
		return 0, err
	}
	if choice == customDuration {
		return parseBenchmarkDuration(custom)
	}
	return parseBenchmarkDuration(choice)
}

func parseBenchmarkDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, errors.New("not a duration, use something like 90s, 10m or 1h")
	}
	if d <= 0 {
		return 0, errors.New("duration must be positive")
	}
	return d, nil
}

func logProgress(ctx context.Context, run *bench.Run) func() {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(progressInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ui.Info("generating uuids", "generated", humanize.Comma(int64(run.Generated.Load())))
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}
