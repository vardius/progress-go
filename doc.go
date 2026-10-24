// Package progress draws a progress bar for a known amount of work.
//
//	bar := progress.New(int64(len(files)), progress.Options{})
//	bar.Start()
//	for _, f := range files {
//		process(f)
//		bar.Advance(1)
//	}
//	if err := bar.Stop(); err != nil {
//		return err
//	}
//
// The bar is fifty cells wide and shows the count, the bar and the percentage:
//
//	35/50 [███████████████████████████████████               ]  70%
//
// A verbose bar adds the heap size, the time elapsed, the estimated total time and the time remaining:
//
//	35/50 [███████████████████████████████████               ]  70% 2.1 MB 35s/50s 15s
//
// # Output
//
// On a terminal, Start draws the bar and a timer redraws it in place every 100 ms until Stop, which draws
// it one last time and ends the line. A verbose bar is coloured. When Output is anything else, such as a
// file, a pipe or a buffer, the bar writes no escape codes and no intermediate frames: Stop writes it once,
// as a plain line. Output is a terminal when it is an *os.File for a character device.
//
// While a bar runs on a terminal, write nothing else to it: the next redraw would overwrite the line.
//
// # Counting
//
// Advance adds to the work done, from any number of goroutines. The percentage and the bar are rounded down
// and kept within 0 and 100%, while the count shows the true number, so work past the total or below zero is
// still visible. A total of 0 or less is complete from the start. Stop draws the bar as it stands: a job that
// fails at 3 of 10 ends showing 3 of 10.
//
// To resume a job, Advance by the work already done before calling Start. The time estimates assume that
// the rate of work since Start holds, so earlier work doesn't make them optimistic.
//
// # Cost
//
// Advance is one atomic add: it never draws and allocates nothing. Each redraw formats the line into a
// reused buffer with the strconv append functions and makes a single Write, allocating nothing. A verbose
// bar reads the heap size from runtime/metrics, which, unlike runtime.ReadMemStats, doesn't stop the world.
// New allocates the bar and its buffers, plus, for an *os.File, the file information it checks for a
// terminal. Start, on a terminal, allocates its timer.
package progress
