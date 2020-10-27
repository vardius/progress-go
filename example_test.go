package progress_test

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/vardius/progress-go"
)

func Example() {
	const totalIterations = 10

	var b bytes.Buffer
	bar := progress.New(0, totalIterations, progress.Options{
		// Verbose: true,
		Output: io.Writer(&b),
		Graph:  "#",
	})

	_, _ = bar.Start()

	for i := 0; i < totalIterations; i++ {
		_, _ = bar.Advance(1)
	}

	if _, err := bar.Stop(); err != nil {
		log.Printf("failed to finish progress: %v", err)
	}

	fmt.Println(b.String())
	//      10/10 [##################################################] 100%
}
