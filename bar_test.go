package progress_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/vardius/progress-go"
)

// clearLine starts every frame drawn to a terminal.
const clearLine = "\r\x1b[K"

// hashes is a bar of '#' with n of its 50 cells filled.
func hashes(n int) string {
	return "[" + strings.Repeat("#", n) + strings.Repeat(" ", 50-n) + "]"
}

// screen is a buffer that a test can read while a bar's redraws write to it.
type screen struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *screen) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *screen) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// lastFrame returns the last frame drawn to a terminal.
func lastFrame(s string) string {
	return s[strings.LastIndex(s, clearLine):]
}

// percentTail returns what a line shows after its percentage: a verbose bar's heap size and times.
func percentTail(t *testing.T, line string) string {
	t.Helper()
	_, tail, ok := strings.Cut(line, "% ")
	if !ok {
		t.Fatalf("no percentage in %q", line)
	}
	return tail
}

func TestStop(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		steps []int64
		want  string
	}{
		{"nothing done", 10, nil, " 0/10 " + hashes(0) + "   0%\n"},
		{"half", 10, []int64{5}, " 5/10 " + hashes(25) + "  50%\n"},
		// Regression: with a one-byte glyph, the untagged version's bar ran to 83 cells at Stop.
		{"done", 10, []int64{10}, "10/10 " + hashes(50) + " 100%\n"},
		{"rounds down", 200, []int64{199}, "199/200 " + hashes(49) + "  99%\n"},
		// Regression: the bar was built a cell at a time and fell behind the percentage.
		{"uneven steps", 100, []int64{1, 3, 2}, "  6/100 " + hashes(3) + "   6%\n"},
		// Regression: Stop filled the bar, so a job that failed part way looked done.
		{"stopped early", 10, []int64{3}, " 3/10 " + hashes(15) + "  30%\n"},
		// Regression: the percentage went past 100 and the bar past its end.
		{"past the total", 10, []int64{30}, "30/10 " + hashes(50) + " 100%\n"},
		// Regression: the percentage went below 0.
		{"below zero", 10, []int64{-5}, "-5/10 " + hashes(0) + "   0%\n"},
		{"far below zero", 100, []int64{-15}, "-15/100 " + hashes(0) + "   0%\n"},
		// Regression: a total of 0 showed 0% beside a full bar, 83 cells long by Stop.
		{"zero total", 0, nil, "0/0 " + hashes(50) + " 100%\n"},
		{"negative total", -3, []int64{1}, "1/0 " + hashes(50) + " 100%\n"},
		// Regression: float32 maths rounded 1099511627775 of 1099511627776 up to 100%.
		{"large total", 1 << 40, []int64{1<<40 - 1}, "1099511627775/1099511627776 " + hashes(49) + "  99%\n"},
		{"largest total", math.MaxInt64, []int64{math.MaxInt64 - 1}, "9223372036854775806/9223372036854775807 " + hashes(49) + "  99%\n"},
		{"largest total, done", math.MaxInt64, []int64{math.MaxInt64}, "9223372036854775807/9223372036854775807 " + hashes(50) + " 100%\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			bar := progress.New(tt.total, progress.Options{Output: &out, Graph: "#"})
			bar.Start()
			for _, n := range tt.steps {
				bar.Advance(n)
			}
			if err := bar.Stop(); err != nil {
				t.Fatalf("Stop() = %v", err)
			}

			if got := out.String(); got != tt.want {
				t.Errorf("output\n got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestGraph(t *testing.T) {
	tests := []struct {
		name, graph string
		half        string // the bar at 1 of 2
	}{
		{"default", "", "[" + strings.Repeat("█", 25) + strings.Repeat(" ", 25) + "]"},
		{"one byte", "=", "[" + strings.Repeat("=", 25) + strings.Repeat(" ", 25) + "]"},
		{"two bytes", "é", "[" + strings.Repeat("é", 25) + strings.Repeat(" ", 25) + "]"},
		{"four bytes", "🟩", "[" + strings.Repeat("🟩", 25) + strings.Repeat(" ", 25) + "]"},
		{"two runes", "=>", "[" + strings.Repeat("=>", 25) + strings.Repeat(" ", 50) + "]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			bar := progress.New(2, progress.Options{Output: &out, Graph: tt.graph})
			bar.Advance(1)
			_ = bar.Stop()

			if want := "1/2 " + tt.half + "  50%\n"; out.String() != want {
				t.Errorf("output\n got %q\nwant %q", out.String(), want)
			}
		})
	}
}

// Regression: the untagged version assumed a three-byte glyph, so other glyphs changed the bar's width.
func TestGraphKeepsWidth(t *testing.T) {
	for _, graph := range []string{"#", "█", "é", "🟩", "=>"} {
		var widths []int
		for done := range int64(11) {
			var out bytes.Buffer
			bar := progress.New(10, progress.Options{Output: &out, Graph: graph})
			bar.Advance(done)
			_ = bar.Stop()

			line := out.String()
			inside := line[strings.Index(line, "[")+1 : strings.LastIndex(line, "]")]
			widths = append(widths, len([]rune(inside)))
		}
		for _, w := range widths {
			if w != widths[0] {
				t.Errorf("Graph %q: bar widths %v, want them all equal", graph, widths)
				break
			}
		}
	}
}

// Regression: escape codes went to files and buffers, and every Advance wrote a frame.
func TestNonTerminalWritesOnePlainLine(t *testing.T) {
	var out bytes.Buffer
	bar := progress.New(10, progress.Options{Output: &out, Verbose: true})
	bar.Start()
	bar.Advance(5)
	if out.Len() != 0 {
		t.Fatalf("Start and Advance wrote %q, want nothing before Stop", out.String())
	}
	_ = bar.Stop()

	if got := out.String(); strings.Count(got, "\n") != 1 || !strings.HasSuffix(got, "\n") || strings.Contains(got, "\x1b") {
		t.Errorf("output %q, want one line with no escape codes", got)
	}
}

func TestTerminal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out screen
		bar := progress.NewTerminal(4, progress.Options{Output: &out, Graph: "#"})
		bar.Start()
		bar.Advance(1)
		time.Sleep(100 * time.Millisecond) // the first redraw
		synctest.Wait()
		bar.Advance(1)
		time.Sleep(50 * time.Millisecond) // too soon for the next one
		bar.Advance(1)
		time.Sleep(50 * time.Millisecond) // the second redraw
		synctest.Wait()
		bar.Advance(1)
		if err := bar.Stop(); err != nil {
			t.Fatalf("Stop() = %v", err)
		}

		want := clearLine + "0/4 " + hashes(0) + "   0%" +
			clearLine + "1/4 " + hashes(12) + "  25%" +
			clearLine + "3/4 " + hashes(37) + "  75%" +
			clearLine + "4/4 " + hashes(50) + " 100%\n"
		if got := out.String(); got != want {
			t.Errorf("output\n got %q\nwant %q", got, want)
		}
	})
}

// Regression: every Advance redrew the line, so a tight loop flooded the terminal.
func TestTerminalRedrawsEvery100ms(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out screen
		bar := progress.NewTerminal(1_000_000, progress.Options{Output: &out})
		bar.Start()
		for range 1_000_000 {
			bar.Advance(1)
		}
		synctest.Wait()
		if n := strings.Count(out.String(), clearLine); n != 1 {
			t.Errorf("drew %d frames for a million Advances in no time, want 1", n)
		}

		time.Sleep(time.Second)
		synctest.Wait()
		if n := strings.Count(out.String(), clearLine); n != 11 {
			t.Errorf("drew %d frames by 1s, want 11: Start's and one every 100 ms", n)
		}
		_ = bar.Stop()
	})
}

// Regression: the reset code was "\x1b[0;1m", which turns bold on, so the terminal stayed bold after the bar.
func TestTerminalVerbose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out screen
		bar := progress.NewTerminal(10, progress.Options{Output: &out, Graph: "#", Verbose: true})
		bar.SetHeap(1 << 20)
		bar.Start()
		start := out.String()
		bar.Advance(5)
		time.Sleep(time.Second)
		synctest.Wait()
		_ = bar.Stop()

		if want := clearLine + " 0/10 \x1b[32;1m" + hashes(0) + "\x1b[0m   0% \x1b[33;1m1.0 MB\x1b[0m 0s/? ?"; start != want {
			t.Errorf("Start drew\n %q\nwant %q", start, want)
		}
		if want := clearLine + " 5/10 \x1b[32;1m" + hashes(25) + "\x1b[0m  50% \x1b[33;1m1.0 MB\x1b[0m 1s/2s 1s\n"; lastFrame(out.String()) != want {
			t.Errorf("Stop drew\n %q\nwant %q", lastFrame(out.String()), want)
		}
	})
}

func TestVerboseTimes(t *testing.T) {
	tests := []struct {
		name    string
		total   int64
		before  int64         // advanced before Start
		wait    time.Duration // between Start and the last Advance
		advance int64
		want    string // after the heap size
	}{
		{"nothing done yet", 10, 0, time.Second, 0, "1s/? ?"},
		{"a tenth in a second", 10, 0, time.Second, 1, "1s/10s 9s"},
		{"done", 10, 0, time.Second, 10, "1s/1s 0s"},
		{"past the total", 10, 0, time.Second, 12, "1s/1s 0s"},
		{"nothing to do", 0, 0, time.Second, 0, "1s/1s 0s"},
		{"nothing to do, below zero", 0, 0, time.Second, -1, "1s/1s 0s"},
		// Regression: the work left, total less done, overflowed and read as done.
		{"below zero, the largest total", math.MaxInt64, 0, time.Second, -1, "1s/? ?"},
		{"resumed at the total", 10, 10, time.Second, 0, "1s/1s 0s"},
		{"no time yet", 10, 0, 0, 1, "0s/? ?"},
		// Only the work since Start sets the rate: 10 in a second leaves 40 for 4s.
		{"resumed", 100, 50, time.Second, 10, "1s/5s 4s"},
		// Regression: the estimate divided whole nanoseconds per unit first, so above a unit a nanosecond it was 0.
		{"more than a unit a nanosecond", 6e9, 0, time.Second, 3e9, "1s/2s 1s"},
		{"rounds down", 2, 0, 1499 * time.Millisecond, 1, "1s/3s 1s"},
		{"rounds up", 2, 0, 1500 * time.Millisecond, 1, "2s/3s 2s"},
		{"minutes", 2, 0, 90 * time.Second, 1, "1m30s/3m0s 1m30s"},
		{"hours", 2, 0, time.Hour + 5*time.Second, 1, "1h0m5s/2h0m10s 1h0m5s"},
		{"days", 2, 0, 25 * time.Hour, 1, "1d1h0m0s/2d2h0m0s 1d1h0m0s"},
		{"years", 2, 0, 400 * 24 * time.Hour, 1, "1y35d0s/2y70d0s 1y35d0s"},
		// 1s a unit for the rest of math.MaxInt64 units is beyond a Duration: capped at the longest one.
		{"beyond a Duration", math.MaxInt64, 0, time.Second, 1, "1s/292y171d23h47m16s 292y171d23h47m16s"},
		// 2^33 ns a unit for 2^30 units is 2^63 ns, one past the longest Duration.
		{"just beyond a Duration", 1<<30 + 1, 0, 1 << 33, 1, "9s/292y171d23h47m16s 292y171d23h47m16s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var out bytes.Buffer
				bar := progress.New(tt.total, progress.Options{Output: &out, Verbose: true})
				bar.SetHeap(0)
				bar.Advance(tt.before)
				bar.Start()
				time.Sleep(tt.wait)
				bar.Advance(tt.advance)
				_ = bar.Stop()

				if got, want := percentTail(t, out.String()), "0 B "+tt.want+"\n"; got != want {
					t.Errorf("after the percentage got %q, want %q", got, want)
				}
			})
		})
	}
}

func TestVerboseNeverStarted(t *testing.T) {
	var out bytes.Buffer
	bar := progress.New(10, progress.Options{Output: &out, Verbose: true})
	bar.SetHeap(0)
	bar.Advance(1)
	_ = bar.Stop()

	if got, want := percentTail(t, out.String()), "0 B 0s/? ?\n"; got != want {
		t.Errorf("after the percentage got %q, want %q", got, want)
	}
}

func TestVerboseHeap(t *testing.T) {
	tests := []struct {
		bytes           uint64
		plain, coloured string
	}{
		{0, "0 B", "0 B"},
		{1023, "1023 B", "1023 B"},
		{1 << 10, "1 KB", "\x1b[36;1m1 KB\x1b[0m"},
		{1536, "2 KB", "\x1b[36;1m2 KB\x1b[0m"},
		{1 << 20, "1.0 MB", "\x1b[33;1m1.0 MB\x1b[0m"},
		{3 << 19, "1.5 MB", "\x1b[33;1m1.5 MB\x1b[0m"},
		{1 << 30, "1.0 GB", "\x1b[31;1m1.0 GB\x1b[0m"},
		{3 << 29, "1.5 GB", "\x1b[31;1m1.5 GB\x1b[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.plain, func(t *testing.T) {
			var plain, tty bytes.Buffer
			for _, bar := range []*progress.Bar{
				progress.New(1, progress.Options{Output: &plain, Verbose: true}),
				progress.NewTerminal(1, progress.Options{Output: &tty, Verbose: true}),
			} {
				bar.SetHeap(tt.bytes)
				_ = bar.Stop()
			}

			if got := percentTail(t, plain.String()); !strings.HasPrefix(got, tt.plain+" 0s/") {
				t.Errorf("plain: after the percentage got %q, want %q first", got, tt.plain)
			}
			if got := percentTail(t, tty.String()); !strings.HasPrefix(got, tt.coloured+" 0s/") {
				t.Errorf("terminal: after the percentage got %q, want %q first", got, tt.coloured)
			}
		})
	}
}

// The real heap size, from runtime/metrics.
func TestVerboseReadsHeap(t *testing.T) {
	var out bytes.Buffer
	bar := progress.New(1, progress.Options{Output: &out, Verbose: true})
	_ = bar.Stop()

	size, _, _ := strings.Cut(percentTail(t, out.String()), " 0s/")
	n, unit, _ := strings.Cut(size, " ")
	if n == "0" || !strings.Contains("B KB MB GB", unit) {
		t.Errorf("heap size %q, want a positive size in B, KB, MB or GB", size)
	}
}

func TestStopWithoutStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out screen
		bar := progress.NewTerminal(2, progress.Options{Output: &out, Graph: "#"})
		bar.Advance(1)
		_ = bar.Stop()
		time.Sleep(time.Second)

		if want := clearLine + "1/2 " + hashes(25) + "  50%\n"; out.String() != want {
			t.Errorf("output\n got %q\nwant %q", out.String(), want)
		}
	})
}

func TestStartTwice(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out screen
		bar := progress.NewTerminal(2, progress.Options{Output: &out})
		bar.Start()
		time.Sleep(50 * time.Millisecond)
		bar.Start() // neither draws nor moves the next redraw
		synctest.Wait()
		if n := strings.Count(out.String(), clearLine); n != 1 {
			t.Errorf("drew %d frames by 50 ms, want 1", n)
		}

		time.Sleep(50 * time.Millisecond)
		synctest.Wait()
		if n := strings.Count(out.String(), clearLine); n != 2 {
			t.Errorf("drew %d frames by 100 ms, want 2", n)
		}
		_ = bar.Stop()
	})
}

// Stop can take the lock just as the timer fires. The redraw that then runs must not draw after Stop's line.
func TestStopAsTheTimerFires(t *testing.T) {
	for range 200 { // the order of the two is up to the scheduler
		synctest.Test(t, func(t *testing.T) {
			var out screen
			bar := progress.NewTerminal(2, progress.Options{Output: &out})
			bar.Start()
			time.Sleep(100 * time.Millisecond) // the timer fires now
			if err := bar.Stop(); err != nil {
				t.Fatalf("Stop() = %v", err)
			}
			stopped := out.String()

			synctest.Wait()
			if out.String() != stopped {
				t.Fatalf("wrote %q after Stop, want nothing", strings.TrimPrefix(out.String(), stopped))
			}
		})
	}
}

func TestAfterStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out screen
		bar := progress.NewTerminal(2, progress.Options{Output: &out})
		bar.Start()
		if err := bar.Stop(); err != nil {
			t.Fatalf("Stop() = %v", err)
		}
		stopped := out.String()

		bar.Advance(1)
		bar.Start()
		time.Sleep(time.Second)
		if err := bar.Stop(); err != nil {
			t.Errorf("second Stop() = %v", err)
		}

		if out.String() != stopped {
			t.Errorf("wrote %q after Stop, want nothing", strings.TrimPrefix(out.String(), stopped))
		}
	})
}

func TestStopFromManyGoroutines(t *testing.T) {
	var out bytes.Buffer
	bar := progress.New(2, progress.Options{Output: &out})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { _ = bar.Stop() })
	}
	wg.Wait()

	if n := strings.Count(out.String(), "\n"); n != 1 {
		t.Errorf("wrote %d lines, want 1", n)
	}
}

// Regression: Advance from several goroutines raced on the count and the bar.
func TestAdvanceFromManyGoroutines(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		var out bytes.Buffer
		bar := progress.New(8000, progress.Options{Output: &out, Graph: "#"})
		bar.Start()
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 1000 {
					bar.Advance(1)
				}
			})
		}
		wg.Wait()
		_ = bar.Stop()

		if want := "8000/8000 " + hashes(50) + " 100%\n"; out.String() != want {
			t.Errorf("output\n got %q\nwant %q", out.String(), want)
		}
	})

	// Redraws run while the goroutines advance, so the race detector sees them side by side.
	t.Run("terminal", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var out screen
			bar := progress.NewTerminal(400, progress.Options{Output: &out, Graph: "#", Verbose: true})
			bar.Start()
			var wg sync.WaitGroup
			for range 4 {
				wg.Go(func() {
					for range 100 {
						bar.Advance(1)
						time.Sleep(10 * time.Millisecond)
					}
				})
			}
			wg.Wait()
			_ = bar.Stop()

			if want := clearLine + "400/400 \x1b[32;1m" + hashes(50) + "\x1b[0m 100%"; !strings.HasPrefix(lastFrame(out.String()), want) {
				t.Errorf("Stop drew\n %q\nwant it to start %q", lastFrame(out.String()), want)
			}
		})
	})
}

var errBroken = errors.New("broken pipe")

// failAfter is a writer whose writes fail after the first ok.
type failAfter struct {
	ok, writes int
}

func (w *failAfter) Write(p []byte) (int, error) {
	if w.writes++; w.writes > w.ok {
		return 0, errBroken
	}
	return len(p), nil
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		terminal   bool
		ok         int
		wantWrites int
	}{
		{"plain: Stop's write", false, 0, 1},
		{"terminal: Start's write", true, 0, 1},
		{"terminal: a redraw", true, 1, 2},
		{"terminal: Stop's write", true, 11, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := &failAfter{ok: tt.ok}
				newBar := progress.New
				if tt.terminal {
					newBar = progress.NewTerminal
				}
				bar := newBar(2, progress.Options{Output: w})
				bar.Start()
				time.Sleep(time.Second) // ten redraws, unless one fails
				synctest.Wait()
				err := bar.Stop()

				if !errors.Is(err, errBroken) || !strings.HasPrefix(err.Error(), "progress: ") {
					t.Errorf("Stop() = %v, want %v with the package's prefix", err, errBroken)
				}
				if again := bar.Stop(); again != err {
					t.Errorf("second Stop() = %v, want %v again", again, err)
				}
				if w.writes != tt.wantWrites {
					t.Errorf("tried %d writes, want %d: none after the first failure", w.writes, tt.wantWrites)
				}
			})
		})
	}
}

func TestIsTerminal(t *testing.T) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	file, err := os.Create(filepath.Join(t.TempDir(), "log"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	closed, err := os.Create(filepath.Join(t.TempDir(), "closed"))
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()

	tests := []struct {
		name string
		w    io.Writer
		want bool
	}{
		{"character device", devNull, true},
		{"pipe", w, false},
		{"file", file, false},
		{"closed file", closed, false},
		{"buffer", &bytes.Buffer{}, false},
	}
	for _, tt := range tests {
		if got := progress.IsTerminal(tt.w); got != tt.want {
			t.Errorf("%s: IsTerminal() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
