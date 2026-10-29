package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
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
			ui.Info("generating uuids", "duration", duration, "workers", run.Workers)
			stop := logProgress(cmd.Context(), run)
			result := run.Start(cmd.Context())
			stop()

			printBenchmark(cmd.OutOrStdout(), result, run.Bucket, capacity)
			return nil
		},
	}
	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "how long to generate uuids (1m, 5m or any duration like 90s)")
	return cmd
}

func askDuration() (time.Duration, error) {
	choice := time.Minute.String()
	custom := ""
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How long should the benchmark run?").
				Options(
					huh.NewOption("1 minute", time.Minute.String()),
					huh.NewOption("5 minutes", (5*time.Minute).String()),
					huh.NewOption("custom", customDuration),
				).
				Value(&choice),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Duration").
				Placeholder("90s, 10m, 1h").
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

func printBenchmark(w io.Writer, result bench.Result, bucket *bench.Bucket, capacity bench.Capacity) {
	rate := result.Rate()
	set := bucket.Set()
	rows := [][2]string{
		{"uuids generated", fmt.Sprintf("%s in %s (%s/s on %d workers)",
			humanize.Comma(int64(result.Generated)),
			result.Elapsed.Round(time.Millisecond),
			humanize.Comma(int64(rate)),
			result.Workers)},
		{"index found", fmt.Sprintf("%s uuids landed on an index already set to 1",
			humanize.Comma(int64(result.Found)))},
		{"bucket", fmt.Sprintf("%s, %s indexes of %d base64 chars, %.2f%% set to 1",
			humanize.IBytes(bucket.Bytes()),
			humanize.Comma(int64(bucket.Indexes())),
			bench.IndexChars,
			100*float64(set)/float64(bucket.Indexes()))},
		{"find every uuid", bench.FormatSeconds(bench.SecondsToFindAll(rate))},
		{"fill memory", fmt.Sprintf("%s of ram in %s",
			humanize.IBytes(capacity.Memory),
			bench.FormatSeconds(bench.SecondsToFill(capacity.Memory, rate)))},
		{"fill storage", fmt.Sprintf("%s free on disk in %s",
			humanize.IBytes(capacity.Disk),
			bench.FormatSeconds(bench.SecondsToFill(capacity.Disk, rate)))},
	}
	for _, row := range rows {
		fmt.Fprintf(w, "%s %s\n", ui.Title.Render(fmt.Sprintf("%-16s", row[0])), row[1])
	}
}
