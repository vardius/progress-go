package progress_test

import (
	"fmt"
	"sync"

	"github.com/vardius/progress-go"
)

func Example() {
	files := []string{"a.txt", "b.txt", "c.txt", "d.txt"}

	// Output isn't a terminal here, so Stop writes the bar once, as a plain line.
	bar := progress.New(int64(len(files)), progress.Options{Graph: "#"})
	bar.Start()
	for range files {
		// Process the file.
		bar.Advance(1)
	}
	if err := bar.Stop(); err != nil {
		fmt.Println(err)
	}
	// Output: 4/4 [##################################################] 100%
}

func ExampleBar_Advance() {
	bar := progress.New(1000, progress.Options{Graph: "#"})
	bar.Start()

	// Four workers share the bar: Advance is safe from any goroutine.
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 250 {
				bar.Advance(1)
			}
		})
	}
	wg.Wait()

	_ = bar.Stop()
	// Output: 1000/1000 [##################################################] 100%
}

func ExampleBar_Stop() {
	bar := progress.New(10, progress.Options{Graph: "#"})
	bar.Start()
	bar.Advance(3)

	// The job fails at 3 of 10: Stop shows where it stopped.
	_ = bar.Stop()
	// Output: 3/10 [###############                                   ]  30%
}

func ExampleOptions() {
	bar := progress.New(2, progress.Options{Graph: "=>"})
	bar.Advance(1)

	// An empty cell is as wide as Graph, so the bar keeps its width.
	_ = bar.Stop()
	// Output: 1/2 [=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>=>                                                  ]  50%
}
