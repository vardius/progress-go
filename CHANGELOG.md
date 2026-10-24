# Changelog

## v1.0.0 (2026-10-24)

The first tagged release. The bar looks as before; underneath it is rewritten to be cheap to advance, safe from
many goroutines and correct at the edges. See [MIGRATION.md](MIGRATION.md) for the before and after of every call.

### Added

- `Advance` is safe from many goroutines at once.
- On a terminal, a timer redraws the bar every 100 ms from `Start` until `Stop`, so a tight loop no longer floods
  the terminal, and a slow job's times keep ticking.
- When `Output` isn't a terminal, the bar writes no escape codes: `Stop` writes it once, as a plain line.
- `Stop` returns the first error from writing to `Output`.
- Runnable examples in `examples/`, each with a test: counting steps, copying a file, checksumming files on
  several goroutines.

### Changed

- `New(total, opts)` replaces `New(start, max, opts...)`. To start part way, `Advance` before `Start`; the time
  estimates then count only the work done since `Start`.
- `Start` and `Advance` return nothing, and `Stop` returns an `error`.
- `Advance` is one atomic add, about 2 ns, and never draws or allocates. Each redraw formats into a reused buffer
  and makes one `Write`, allocating nothing. A bar of 1,000 updates on a terminal went from 937 µs and 2,781
  allocations to 4.26 µs and 6. Allocation counts are guarded by tests.
- A verbose bar reads the heap size from `runtime/metrics`, which doesn't stop the world as
  `runtime.ReadMemStats` did on every update. It is close to the number before, and can read lower.
- `Stop` draws the bar as it stands. It no longer fills the bar, so a job that fails part way doesn't look done.
- The count is padded to the width of the total, not to 8 characters.
- The estimated and remaining times show `?` until some work is done, not `0s`.
- Go 1.26 or newer is required.

### Fixed

- The bar fell behind the percentage after uneven steps, and with a one-byte `Graph` it ran past its 50 cells
  at `Stop`. Any glyph now fills exactly 50 cells, and an empty cell is as wide as `Graph`.
- The percentage went above 100 or below 0, and the bar past its end; both now stay within 0 and 100%.
- A total of 0 showed 0% beside a full bar; it now shows 100%.
- float32 maths rounded large counts: 1,099,511,627,775 of 1,099,511,627,776 showed 100%. The maths is now exact
  for any `int64`.
- The estimates divided whole nanoseconds per unit first, so above a unit a nanosecond they were 0.
- The colour reset code also turned bold on, leaving the terminal bold after a verbose bar.

### Removed

- Travis CI and Hound, replaced by GitHub Actions.
