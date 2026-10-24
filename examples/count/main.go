// Command count counts to ten, a tenth of a second at a time, under a verbose bar.
//
//	go run ./examples/count
package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/vardius/progress-go"
)

// count does n steps of work, each taking the given time, and advances the bar after each.
func count(out io.Writer, n int, each time.Duration) error {
	bar := progress.New(int64(n), progress.Options{Output: out, Verbose: true})
	bar.Start()
	for range n {
		time.Sleep(each) // a step of real work goes here
		bar.Advance(1)
	}
	return bar.Stop()
}

func main() {
	log.SetFlags(0)
	if err := count(os.Stdout, 10, 100*time.Millisecond); err != nil {
		log.Fatal(err)
	}
}
