package progress_test

import (
	"io"
	"testing"

	"github.com/vardius/progress-go"
)

// The bars draw to io.Discard, so what is measured is the package's own cost, not the terminal's.

func BenchmarkAdvance(b *testing.B) {
	bar := progress.New(1<<62, progress.Options{Output: io.Discard})
	b.ReportAllocs()
	for b.Loop() {
		bar.Advance(1)
	}
}

// Every goroutine advances the same bar: the cost of sharing its counter.
func BenchmarkAdvanceParallel(b *testing.B) {
	bar := progress.New(1<<62, progress.Options{Output: io.Discard})
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bar.Advance(1)
		}
	})
}

// One frame, as the timer draws it ten times a second on a terminal.
func BenchmarkRedraw(b *testing.B) {
	bar := progress.NewTerminal(1<<40, progress.Options{Output: io.Discard})
	b.ReportAllocs()
	for b.Loop() {
		bar.Advance(1 << 20)
		bar.Draw()
	}
}

// One verbose frame: the heap size from runtime/metrics, and the times.
func BenchmarkRedrawVerbose(b *testing.B) {
	bar := progress.NewTerminal(1<<40, progress.Options{Output: io.Discard, Verbose: true})
	bar.Start()
	defer bar.Stop()
	b.ReportAllocs()
	for b.Loop() {
		bar.Advance(1 << 20)
		bar.Draw()
	}
}

// A whole bar on a terminal: New, Start, a thousand Advances and Stop.
func BenchmarkBar(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		bar := progress.NewTerminal(1000, progress.Options{Output: io.Discard})
		bar.Start()
		for range 1000 {
			bar.Advance(1)
		}
		_ = bar.Stop()
	}
}
