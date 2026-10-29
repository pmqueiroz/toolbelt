# tb benchmark

Generates random v4 uuids on every core for a fixed time and turns the rate
into "how long would it take" estimates. Code: `internal/bench/`,
`internal/cli/benchmark.go`.

## Duration

`--duration/-d` takes any Go duration (`5s`, `10s`, `1m`). Without the flag
and with a TTY, a huh select offers 5 seconds, 10 seconds or custom (a
validated input). Without a TTY and no flag it runs the 5s default.
Ctrl+C stops early; the results still print, computed from the real elapsed
time.

## The indexed bucket

Full uuids are never stored. Each uuid is base64 encoded (raw url alphabet,
22 chars) and the first `IndexChars` (5) chars become the index: each char is
6 bits, so 5 chars address 2^30 indexes. The value is one bit: 0 not found, 1
found. The bucket is a `[]atomic.Uint64` bitset of 128 MiB, marked with
`atomic.Uint64.Or` so all workers share it without a lock. `Mark` returns
whether the bit was already 1; those uuids are counted as "index found".

Why a prefix and not the whole uuid: a bit per possible uuid would need
2^122 bits. The prefix keeps memory fixed, at the price that "found" means
"a uuid with the same 30-bit prefix was seen", not "the same uuid was seen".
At ~50M uuids/s the bucket saturates within a minute; the fill percentage
shows that.

Indexing goes through the base64 string on purpose (it is the index the
feature asked for); it costs little next to `crypto/rand`.

## Generation

One worker per `runtime.NumCPU()`, each filling batches of 1024 uuids from
`crypto/rand` and setting the v4 version nibble and RFC 4122 variant bits.
Counters are shared atomics.

## Display

With a TTY, a bubbletea program (`internal/cli/benchmark_view.go`) redraws on
stderr every 200ms: spinner, a time bar, the live count and rate, and a
bucket fill bar (`Bucket.Set` popcounts 128 MiB per refresh, a few ms). The
run itself happens in a goroutine; Ctrl+C arrives as a key (raw mode) and
cancels it, and the program quits when the run reports done, so the partial
result is never lost. Without a TTY the same run logs a start line and the
running count every 15s instead.

The results print to stdout inside `ui.Panel`: run stats, the bucket bar,
then one `ui.Bar` per estimate on a log10 scale relative to the longest one.
Log scale because the values span ~30 orders of magnitude (seconds to fill
RAM vs 10^23 years to find every uuid); a linear bar would show only the
last one.

## Estimates

All from `rate = generated / elapsed`:

- **find every uuid**: coupon collector, `N·(ln N + γ) / rate` with
  `N = 2^122` (the random bits of a v4 uuid). Drawing randomly, this is the
  expected number of draws to see every value at least once, not `N`.
- **fill memory / storage**: `capacity / (16 B · rate)`, i.e. if every
  generated uuid were kept as its raw 16 bytes. Memory is total physical RAM
  (`sysinfo` on linux, `hw.memsize` on macOS, `GlobalMemoryStatusEx` on
  Windows, called through `kernel32` because `x/sys/windows` has no
  wrapper). Storage is the space available to the user on the volume holding
  the home dir (`statfs` / `GetDiskFreeSpaceEx`).

- **fill earth storage**: same formula against `EarthStorage`, 18 ZB. A
  fixed estimate, not measured: IDC's Global StorageSphere put the world's
  installed storage base at 6.8 ZB in 2020 with a 17.8% yearly growth
  forecast; extrapolated to 2026 that is ~18 ZB. Bump the constant when a
  newer figure is published.

`FormatSeconds` prints durations under a year as Go durations, up to a
millennium as `%.3g years`, and beyond that adds `DescribeYears` in
parentheses: the count in millennia named with the short scale ("292
quintillion millennia"), names up to decillion (larger values keep
growing the number in front of "decillion"). A zero rate prints `never`.

Results go to stdout (`cmd.OutOrStdout()`), progress to stderr.
