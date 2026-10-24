# Migrating from the untagged version

v1.0.0 is the first tagged release. The module path, the `Options` fields and the bar on screen are as before;
the calls are simpler, and a few edge cases behave differently: they are listed below.

## 1. Update the module

```sh
go get github.com/vardius/progress-go@v1.0.0
```

The import path doesn't change. v1 needs Go 1.26 or newer.

## 2. Before and after

```go
// untagged
bar := progress.New(0, int64(len(files)), progress.Options{Verbose: true})
_, _ = bar.Start()
defer func() {
	if _, err := bar.Stop(); err != nil {
		log.Printf("failed to finish progress: %v", err)
	}
}()
for _, f := range files {
	process(f)
	_, _ = bar.Advance(1)
}
```

```go
// v1
bar := progress.New(int64(len(files)), progress.Options{Verbose: true})
bar.Start()
for _, f := range files {
	process(f)
	bar.Advance(1)
}
if err := bar.Stop(); err != nil {
	log.Printf("draw progress: %v", err)
}
```

## 3. Every call

| Untagged | v1 |
| --- | --- |
| `New(start, max int64, opts ...Options) *Bar` | `New(total int64, opts Options) *Bar`; pass `Options{}` for the defaults |
| `New(start, max, ...)` with `start` above 0 | `New(total, ...)`, then `bar.Advance(start)` before `Start` |
| `Options{Output, Graph, Verbose}` | `Options{Output, Graph, Verbose}`, unchanged |
| `(*Bar).Start() (n int, err error)` | `(*Bar).Start()` |
| `(*Bar).Advance(inc int64) (n int, err error)` | `(*Bar).Advance(n int64)` |
| `(*Bar).Stop() (n int, err error)` | `(*Bar).Stop() error` |

The byte counts that `Start`, `Advance` and `Stop` returned are gone. A write error is kept, and `Stop` returns it.

## 4. Behaviour changes

- **Advance doesn't draw.** It adds to the count and returns. On a terminal, `Start` draws the bar and a timer
  redraws it every 100 ms until `Stop`, so call `Stop` when the work ends: it also stops the timer.
- **Advance is safe from many goroutines.** The untagged version needed a lock around it.
- **Output that isn't a terminal gets one plain line.** Files, pipes and buffers get no escape codes and no
  intermediate frames: `Stop` writes the bar once. Output is a terminal when it is an `*os.File` for a character
  device. Tests that read the bar from a `bytes.Buffer` now see just that line.
- **Stop draws the bar as it stands.** The untagged version filled the bar to the total, so a job that failed at
  3 of 10 showed 10 of 10. To show a finished bar, `Advance` to the total first.
- **The count is padded to the width of the total**, so `7/10` is written ` 7/10`, not `       7/10`.
- **The percentage and the bar stay within 0 and 100%**, while the count shows the true number. A total of 0 or
  less shows 100%.
- **Any Graph fills exactly 50 cells**, and an empty cell is a space for each rune of `Graph`, so the bar keeps
  its width. With a one-byte glyph the untagged bar ran past its width: 83 cells for 10 of 10 done in one step.
- **Verbose estimates** show `?` until some work is done, where the untagged version showed `0s`, and they count
  only the work done since `Start`. The heap size is now read from `runtime/metrics`. It is close to the
  `HeapAlloc` the untagged version showed, and can read lower: memory each processor has cached counts only once
  the runtime flushes it.
- **Colours reset to normal text.** The untagged reset code also turned bold on, leaving the terminal bold.
