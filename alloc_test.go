package progress_test

import (
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/vardius/progress-go"
)

// TestAllocations guards the allocations per call. Lower a number when an optimisation lands;
// never raise one without a reason in the commit.
func TestAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes allocation counts")
	}

	plain := progress.New(1<<40, progress.Options{Output: io.Discard})
	terminal := progress.NewTerminal(1<<40, progress.Options{Output: io.Discard})
	verbose := progress.NewTerminal(1<<40, progress.Options{Output: io.Discard, Verbose: true})
	verbose.Start()
	defer verbose.Stop()

	tests := []struct {
		name string
		call func()
		want float64
	}{
		{"Advance", func() { plain.Advance(1) }, 0},
		// A draw formats into the bar's reused buffer and makes one Write.
		{"draw, plain", func() { plain.Advance(1); plain.Draw() }, 0},
		{"draw, terminal", func() { terminal.Advance(1); terminal.Draw() }, 0},
		{"draw, verbose", func() { verbose.Advance(1); verbose.Draw() }, 0},
		// The Bar, the filled bar (Graph 50 times) and the line buffer. The empty bar costs nothing
		// for a Graph of one or two runes: strings.Repeat returns up to 128 spaces from a constant.
		{"New", func() { progress.New(10, progress.Options{Output: io.Discard}) }, 3},
		// Plus the runtime/metrics sample and the function that reads it.
		{"New, verbose", func() { progress.New(10, progress.Options{Output: io.Discard, Verbose: true}) }, 5},
		// New's 3, plus Start's timer and the redraw function it calls.
		{"New, Start and Stop, terminal", func() {
			bar := progress.NewTerminal(10, progress.Options{Output: io.Discard})
			bar.Start()
			_ = bar.Stop()
		}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(1000, tt.call); got != tt.want {
				t.Errorf("allocates %v times per call, want %v", got, tt.want)
			}
		})
	}
}

// TestRedrawAllocations guards a redraw as the timer runs it, verbose, with the heap size read from the runtime.
func TestRedrawAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes allocation counts")
	}

	synctest.Test(t, func(t *testing.T) {
		bar := progress.NewTerminal(1<<40, progress.Options{Output: io.Discard, Verbose: true})
		bar.Start()
		defer bar.Stop()

		got := testing.AllocsPerRun(100, func() {
			bar.Advance(1)
			time.Sleep(100 * time.Millisecond)
			synctest.Wait()
		})
		if got != 0 {
			t.Errorf("a redraw allocates %v times, want 0", got)
		}
	})
}
