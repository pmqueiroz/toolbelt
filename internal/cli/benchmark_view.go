package cli

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/pmqueiroz/toolbelt/internal/bench"
	"github.com/pmqueiroz/toolbelt/internal/ui"
)

const (
	barWidth     = 30
	refreshEvery = 200 * time.Millisecond
	labelWidth   = 16
)

type benchmarkProgress struct {
	Elapsed   time.Duration
	Duration  time.Duration
	Generated uint64
	Set       uint64
	Indexes   uint64
}

type benchmarkTick time.Time

type benchmarkDone struct{}

type benchmarkModel struct {
	run     *bench.Run
	cancel  context.CancelFunc
	spinner spinner.Model
	started time.Time
	now     time.Time
	set     uint64
	done    bool
}

func runBenchmark(ctx context.Context, run *bench.Run) bench.Result {
	if !ui.Interactive() {
		ui.Info("generating uuids", "duration", run.Duration, "workers", run.Workers)
		stop := logProgress(ctx, run)
		defer stop()
		return run.Start(ctx)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	started := time.Now()
	program := tea.NewProgram(benchmarkModel{
		run:     run,
		cancel:  cancel,
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(ui.Pink))),
		started: started,
		now:     started,
	}, tea.WithOutput(os.Stderr))

	results := make(chan bench.Result, 1)
	go func() {
		results <- run.Start(ctx)
		program.Send(benchmarkDone{})
	}()
	if _, err := program.Run(); err != nil {
		cancel()
	}
	return <-results
}

func tickBenchmark() tea.Cmd {
	return tea.Tick(refreshEvery, func(t time.Time) tea.Msg { return benchmarkTick(t) })
}

func (m benchmarkModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tickBenchmark())
}

func (m benchmarkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case benchmarkTick:
		m.now = time.Time(msg)
		m.set = m.run.Bucket.Set()
		return m, tickBenchmark()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancel()
		}
	case benchmarkDone:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m benchmarkModel) View() string {
	if m.done {
		return ""
	}
	return m.spinner.View() + " " + renderBenchmarkProgress(benchmarkProgress{
		Elapsed:   m.now.Sub(m.started),
		Duration:  m.run.Duration,
		Generated: m.run.Generated.Load(),
		Set:       m.set,
		Indexes:   m.run.Bucket.Indexes(),
	}) + "\n"
}

func renderBenchmarkProgress(p benchmarkProgress) string {
	elapsed := min(p.Elapsed, p.Duration)
	timeDone := elapsed.Seconds() / p.Duration.Seconds()
	rate := 0.0
	if p.Elapsed > 0 {
		rate = float64(p.Generated) / p.Elapsed.Seconds()
	}
	bucketDone := float64(p.Set) / float64(p.Indexes)
	return strings.Join([]string{
		ui.Title.Render("generating uuids") + ui.Faint.Render("  ctrl+c to stop early"),
		"  " + row("time", fmt.Sprintf("%s %s / %s", ui.Bar(barWidth, timeDone), elapsed.Round(100*time.Millisecond), p.Duration)),
		"  " + row("generated", fmt.Sprintf("%s · %s/s", humanize.Comma(int64(p.Generated)), humanize.Comma(int64(rate)))),
		"  " + row("bucket", fmt.Sprintf("%s %.1f%% set", ui.Bar(barWidth, bucketDone), 100*bucketDone)),
	}, "\n")
}

func row(label, value string) string {
	return ui.Title.Render(fmt.Sprintf("%-*s", labelWidth, label)) + value
}

func renderBenchmark(result bench.Result, bucket *bench.Bucket, capacity bench.Capacity) string {
	rate := result.Rate()
	set := bucket.Set()
	bucketDone := float64(set) / float64(bucket.Indexes())
	findAll := bench.SecondsToFindAll(rate)
	fillMemory := bench.SecondsToFill(capacity.Memory, rate)
	fillDisk := bench.SecondsToFill(capacity.Disk, rate)
	longest := math.Max(findAll, math.Max(fillMemory, fillDisk))

	stats := strings.Join([]string{
		row("uuids generated", fmt.Sprintf("%s in %s",
			humanize.Comma(int64(result.Generated)),
			result.Elapsed.Round(time.Millisecond))),
		row("rate", fmt.Sprintf("%s/s on %d workers", humanize.Comma(int64(rate)), result.Workers)),
		row("index found", fmt.Sprintf("%s uuids landed on an index already set to 1",
			humanize.Comma(int64(result.Found)))),
		row("bucket", fmt.Sprintf("%s %.2f%% of %s indexes set",
			ui.Bar(barWidth, bucketDone),
			100*bucketDone,
			humanize.Comma(int64(bucket.Indexes())))),
		row("", ui.Faint.Render(fmt.Sprintf("%s, %d base64 chars per index, 1 bit each",
			humanize.IBytes(bucket.Bytes()), bench.IndexChars))),
	}, "\n")

	timeline := strings.Join([]string{
		ui.Faint.Render("how long, log scale"),
		row("fill memory", fmt.Sprintf("%s %s · %s ram",
			ui.Bar(barWidth, logFraction(fillMemory, longest)),
			bench.FormatSeconds(fillMemory),
			humanize.IBytes(capacity.Memory))),
		row("fill storage", fmt.Sprintf("%s %s · %s free",
			ui.Bar(barWidth, logFraction(fillDisk, longest)),
			bench.FormatSeconds(fillDisk),
			humanize.IBytes(capacity.Disk))),
		row("find every uuid", fmt.Sprintf("%s %s",
			ui.Bar(barWidth, logFraction(findAll, longest)),
			bench.FormatSeconds(findAll))),
	}, "\n")

	return ui.Panel("tb benchmark", stats+"\n\n"+timeline)
}

func logFraction(seconds, longest float64) float64 {
	if math.IsInf(longest, 1) || longest <= 1 {
		return 1
	}
	return math.Log10(1+seconds) / math.Log10(1+longest)
}
