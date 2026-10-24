# 💯 progress-go

[![Test](https://github.com/vardius/progress-go/actions/workflows/test.yml/badge.svg)](https://github.com/vardius/progress-go/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/vardius/progress-go.svg)](https://pkg.go.dev/github.com/vardius/progress-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/progress-go)](https://goreportcard.com/report/github.com/vardius/progress-go)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE.md)

A progress bar for Go that stays out of the hot path.

```txt
35/50 [███████████████████████████████████               ]  70% 2.1 MB 35s/50s 15s
```

- **One atomic add.** `Advance` adds to a counter and returns: about 2 ns, no drawing, no allocations.
- **Safe from many goroutines.** Workers can share one bar with no lock of their own.
- **Calm on a terminal.** A timer redraws the bar in place ten times a second, however fast the work goes.
- **Plain in logs.** Written to a file, a pipe or a buffer, the bar is one line at `Stop`, with no escape codes.
- **Exact.** Integer maths for any `int64` count; the percentage stays within 0 and 100%.
- **Verbose** adds the heap size, read from `runtime/metrics` without stopping the world, the time elapsed, the
  estimated total and the time remaining.

📖 **[Documentation](https://vardius.github.io/progress-go/)** · [API reference](https://pkg.go.dev/github.com/vardius/progress-go) · [Examples](examples) · [Migrating from the untagged version](MIGRATION.md)

## Install

```sh
go get github.com/vardius/progress-go
```

Requires Go 1.26 or newer.

## Usage

```go
bar := progress.New(int64(len(files)), progress.Options{})
bar.Start()
for _, f := range files {
	process(f)
	bar.Advance(1)
}
if err := bar.Stop(); err != nil {
	return err
}
```

`Start` starts the clock and, on a terminal, draws the bar and redraws it every 100 ms. `Advance` adds to the work
done, from any goroutine. `Stop` draws the bar as it stands, ends its line and returns the first error from writing it.

Options set where the bar goes, the glyph it fills with, and whether it's verbose:

```go
bar := progress.New(size, progress.Options{
	Output:  os.Stderr, // nil means os.Stdout
	Graph:   "#",       // empty means "█"
	Verbose: true,      // heap size and times
})
```

When `Output` is a terminal, an `*os.File` for a character device, the bar is redrawn in place and a verbose bar is
coloured. Anywhere else it writes no escape codes and no frames along the way: `Stop` writes it once, as a plain line.

## Goroutines

```go
var wg sync.WaitGroup
for range 4 {
	wg.Go(func() {
		for job := range jobs {
			process(job)
			bar.Advance(1)
		}
	})
}
wg.Wait()
err := bar.Stop()
```

See [examples/parallel](examples/parallel/main.go) for a program that checksums files this way, and
[examples/copy](examples/copy/main.go) for one that counts bytes.

## Benchmarks

Time and allocations, on an Apple M4 Max with Go 1.27.0, median of 5 runs. Both versions ran on the same harness.
`Advance` draws to `io.Discard`; a whole bar draws to `/dev/null`, which both versions treat as a terminal, so each
frame costs a write. The untagged `Bar` isn't safe from many goroutines, so there it runs behind a mutex. With 16
goroutines at once, the time is wall time per call across all of them, so it measures throughput.
Run v1's own suite with `go test -bench .`.

| Scenario | Untagged | v1 |
| --- | ---: | ---: |
| Advance | 133 ns · 1 alloc | **1.8 ns · 0 allocs** |
| Advance, verbose | 23.6 µs · 13 allocs | **1.8 ns · 0 allocs** |
| Advance, 16 goroutines at once | 313 ns · 2 allocs | **59.6 ns · 0 allocs** |
| A bar of 1,000 updates, New to Stop | 937 µs · 2,781 allocs | **4.26 µs · 6 allocs** |
| The same, verbose | 25.0 ms · 10,799 allocs | **4.83 µs · 8 allocs** |

The untagged version redrew the line on every `Advance`, and a verbose bar stopped the world to read the heap size
each time. In v1 a redraw takes about 44 ns, or 270 ns verbose, and allocates nothing.

## Upgrading from the untagged version

```go
// untagged
bar := progress.New(0, n, progress.Options{})
_, _ = bar.Start()
_, _ = bar.Advance(1)
_, err := bar.Stop()

// v1
bar := progress.New(n, progress.Options{})
bar.Start()
bar.Advance(1)
err := bar.Stop()
```

`Stop` now shows where the work stopped instead of filling the bar. [MIGRATION.md](MIGRATION.md) maps every call
to its v1 form and lists the behaviour changes.

## Contributing

Bug reports and pull requests are welcome: see [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## License

MIT, see [LICENSE.md](LICENSE.md).
