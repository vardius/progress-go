package progress

import (
	"fmt"
	"io"
	"math"
	"math/bits"
	"os"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// cells is the bar's width: how many Graphs a full bar shows.
const cells = 50

// redrawEvery is how often a running bar on a terminal is redrawn: often enough to look live, and rarely
// enough that the terminal never floods, however fast the work goes.
const redrawEvery = 100 * time.Millisecond

// Terminal escapes. clearLine returns the cursor to the start of the line and clears it.
const (
	clearLine = "\r\x1b[K"
	reset     = "\x1b[0m"
	red       = "\x1b[31;1m"
	green     = "\x1b[32;1m"
	yellow    = "\x1b[33;1m"
	cyan      = "\x1b[36;1m"
)

// heapMetric is the bytes taken by heap objects, as runtime.MemStats.HeapAlloc counts them, read without
// stopping the world as runtime.ReadMemStats does. It can read lower than HeapAlloc: ReadMemStats first
// flushes each processor's cached spans, and the metric counts them only once they are flushed.
const heapMetric = "/memory/classes/heap/objects:bytes"

// Options configure a Bar. The zero value draws to os.Stdout with the default Graph.
type Options struct {
	// Output is where the bar is drawn; nil means os.Stdout. On a terminal the bar is drawn by Start and
	// redrawn in place until Stop. Anywhere else, such as a file, a pipe or a buffer, Stop writes it once,
	// as a plain line.
	Output io.Writer
	// Graph fills one cell of the bar; empty means "█". An empty cell is a space for each rune of Graph,
	// so the bar keeps its width as it fills, as long as each rune takes one column.
	Graph string
	// Verbose adds the heap size, the time elapsed, the estimated total time and the time remaining.
	// On a terminal it also adds colour.
	Verbose bool
}

type state uint8

const (
	created state = iota
	running
	stopped
)

// Bar is a progress bar for a known amount of work. Start it, Advance it as work gets done, and Stop it.
// Its methods are safe to call from many goroutines at once.
type Bar struct {
	// done is the work done so far, the only field Advance touches.
	done atomic.Int64

	total   int64
	out     io.Writer
	tty     bool                  // out is a terminal: redraw in place
	verbose bool                  // add the heap size and times
	full    string                // the bar filled: cells Graphs
	empty   string                // the bar empty: a space for each rune of full
	fill    int                   // bytes of one filled cell
	space   int                   // bytes of one empty cell
	width   int                   // digits in total, so the count keeps its place as it grows
	heap    func() (uint64, bool) // reads the heap size; set when verbose

	mu      sync.Mutex // guards the fields below and orders writes to out
	state   state
	started time.Time
	base    int64 // done at Start: the time estimates count only the work done since
	timer   *time.Timer
	frame   []byte // reused for every draw
	err     error  // the first write error
}

// New returns a Bar for total units of work. A total of 0 or less is complete from the start.
func New(total int64, opts Options) *Bar {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	graph := opts.Graph
	if graph == "" {
		graph = "█"
	}
	space := utf8.RuneCountInString(graph)

	b := &Bar{
		total:   max(total, 0),
		out:     out,
		tty:     isTerminal(out),
		verbose: opts.Verbose,
		full:    strings.Repeat(graph, cells),
		empty:   strings.Repeat(" ", cells*space),
		fill:    len(graph),
		space:   space,
	}
	b.width = digits(b.total)
	// Room for the bar and a verbose line's text, so the first draw doesn't grow the buffer.
	b.frame = make([]byte, 0, len(b.full)+len(b.empty)+128)
	if opts.Verbose {
		b.heap = readHeap()
	}
	return b
}

// Start starts the clock for the time estimates. On a terminal it also draws the bar, and then redraws it
// every 100 ms until Stop. Calls after the first, and after Stop, do nothing.
func (b *Bar) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != created {
		return
	}
	b.state = running
	b.started = time.Now()
	b.base = b.done.Load()
	if !b.tty {
		return
	}
	b.draw("")
	b.timer = time.AfterFunc(redrawEvery, b.redraw)
}

// Advance adds n to the work done; n may be negative. It is one atomic add and never draws, so it is safe
// to call from many goroutines and in tight loops. The bar shows the new count when it is next drawn.
func (b *Bar) Advance(n int64) {
	b.done.Add(n)
}

// Stop draws the bar as it stands and ends its line. A bar stopped before its total shows where it stopped.
// After Stop the bar never writes again. Stop returns the first error from writing to Output; once a write
// fails, the bar stops drawing. Calls after the first draw nothing and return the same error.
func (b *Bar) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == stopped {
		return b.err
	}
	if b.timer != nil {
		b.timer.Stop()
	}
	b.state = stopped
	if b.err == nil {
		b.draw("\n")
	}
	return b.err
}

// redraw is the timer's: it draws the bar and schedules the next redraw, until Stop or a write error.
func (b *Bar) redraw() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != running || b.err != nil {
		return
	}
	b.draw("")
	b.timer.Reset(redrawEvery)
}

// draw writes the bar's line in one Write, ended with end. On a terminal the line first clears the one
// before it. Callers hold b.mu.
func (b *Bar) draw(end string) {
	f := b.frame[:0]
	if b.tty {
		f = append(f, clearLine...)
	}
	f = b.appendLine(f)
	b.frame = append(f, end...)

	if _, err := b.out.Write(b.frame); err != nil {
		b.err = fmt.Errorf("progress: %w", err)
	}
}

// appendLine appends the count, the bar and the percentage, and in a verbose bar the heap size and times:
//
//	35/50 [███████████████████████████████████               ]  70% 2.1 MB 35s/50s 15s
func (b *Bar) appendLine(f []byte) []byte {
	done := b.done.Load()
	colour := b.tty && b.verbose

	f = appendPadded(f, done, b.width)
	f = append(f, '/')
	f = strconv.AppendInt(f, b.total, 10)
	f = append(f, ' ')
	if colour {
		f = append(f, green...)
	}
	filled := int(share(done, b.total, cells))
	f = append(f, '[')
	f = append(f, b.full[:filled*b.fill]...)
	f = append(f, b.empty[:(cells-filled)*b.space]...)
	f = append(f, ']')
	if colour {
		f = append(f, reset...)
	}
	f = append(f, ' ')
	f = appendPadded(f, share(done, b.total, 100), 3)
	f = append(f, '%')
	if !b.verbose {
		return f
	}

	if n, ok := b.heap(); ok {
		f = append(f, ' ')
		f = appendSize(f, n, colour)
	}
	f = append(f, ' ')
	return b.appendTimes(f, done)
}

// appendTimes appends the time elapsed since Start, the estimated total time and the time remaining,
// as elapsed/total remaining. The estimates assume the rate of work since Start holds; until there is a
// rate, some work done in some time, they are unknown and shown as "?".
func (b *Bar) appendTimes(f []byte, done int64) []byte {
	var elapsed time.Duration
	if !b.started.IsZero() {
		elapsed = time.Since(b.started)
	}
	f = appendDuration(f, elapsed)
	f = append(f, '/')

	if b.total == 0 || done >= b.total {
		f = appendDuration(f, elapsed)
		return append(f, " 0s"...)
	}
	if done <= b.base || elapsed <= 0 {
		return append(f, "? ?"...)
	}
	// Differences of int64s that are known to be positive: exact in uint64, where int64 could overflow.
	left, since := uint64(b.total)-uint64(done), uint64(done)-uint64(b.base)
	// In float64: elapsed*left overflows within seconds for a large total.
	remaining := float64(elapsed) / float64(since) * float64(left)
	f = appendDuration(f, toDuration(float64(elapsed)+remaining))
	f = append(f, ' ')
	return appendDuration(f, toDuration(remaining))
}

// share is done's share of total in parts of whole, rounded down and kept within 0..whole.
// A total of 0 is complete. The product is 128-bit, so it is exact for any count.
func share(done, total, whole int64) int64 {
	if total == 0 {
		return whole
	}
	done = min(max(done, 0), total)
	hi, lo := bits.Mul64(uint64(done), uint64(whole))
	q, _ := bits.Div64(hi, lo, uint64(total)) // hi < total, as done <= total
	return int64(q)
}

// toDuration converts nanoseconds to a Duration, capped at the longest one.
func toDuration(ns float64) time.Duration {
	if ns >= math.MaxInt64 {
		return math.MaxInt64
	}
	return time.Duration(ns)
}

// appendDuration appends d rounded to the second, as Duration.String writes it (1h2m3s), after its days
// and years once it has some (1y2d3h4m5s).
func appendDuration(f []byte, d time.Duration) []byte {
	const minute, hour, day, year = 60, 60 * 60, 24 * 60 * 60, 365 * 24 * 60 * 60

	s := int64(d.Round(time.Second) / time.Second)
	if s >= day {
		if s >= year {
			f, s = appendUnit(f, s, year, 'y')
		}
		f, s = appendUnit(f, s, day, 'd')
	}
	if s >= minute {
		if s >= hour {
			f, s = appendUnit(f, s, hour, 'h')
		}
		f, s = appendUnit(f, s, minute, 'm')
	}
	f, _ = appendUnit(f, s, 1, 's')
	return f
}

// appendUnit appends how many whole units s seconds hold, and the unit's name, and returns the seconds left.
func appendUnit(f []byte, s, unit int64, name byte) ([]byte, int64) {
	f = strconv.AppendInt(f, s/unit, 10)
	return append(f, name), s % unit
}

// appendSize appends n bytes in B, KB, MB or GB, of 1024 each. With colour, kilobytes are cyan,
// megabytes yellow and gigabytes red.
func appendSize(f []byte, n uint64, colour bool) []byte {
	v, unit, prec, clr := float64(n), " B", 0, ""
	switch {
	case n >= 1<<30:
		v, unit, prec, clr = v/(1<<30), " GB", 1, red
	case n >= 1<<20:
		v, unit, prec, clr = v/(1<<20), " MB", 1, yellow
	case n >= 1<<10:
		v, unit, prec, clr = v/(1<<10), " KB", 0, cyan
	}

	colour = colour && clr != ""
	if colour {
		f = append(f, clr...)
	}
	f = strconv.AppendFloat(f, v, 'f', prec, 64)
	f = append(f, unit...)
	if colour {
		f = append(f, reset...)
	}
	return f
}

// appendPadded appends n right-aligned in width columns.
func appendPadded(f []byte, n int64, width int) []byte {
	for range width - digits(n) {
		f = append(f, ' ')
	}
	return strconv.AppendInt(f, n, 10)
}

// digits is how many characters n takes in decimal, its sign included.
func digits(n int64) int {
	d := 1
	if n < 0 {
		d++
	}
	for ; n <= -10 || n >= 10; n /= 10 {
		d++
	}
	return d
}

// readHeap returns a reader of the heap size. Its ok is false if the runtime no longer reports it.
func readHeap() func() (uint64, bool) {
	sample := []metrics.Sample{{Name: heapMetric}}
	return func() (n uint64, ok bool) {
		metrics.Read(sample)
		v := sample[0].Value
		if ok = v.Kind() == metrics.KindUint64; ok {
			n = v.Uint64()
		}
		return n, ok
	}
}

// isTerminal reports whether w is a character device, such as a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
