package main

import (
	"log"
	"time"

	"github.com/vardius/progress-go"
)

func main() {
	const start = 0
	const end = 10

	bar := progress.New(start, end, progress.Options{
		Verbose: true,
	})

	_, _ = bar.Start()
	defer func() {
		if _, err := bar.Stop(); err != nil {
			log.Printf("faile to finish progress: %v", err)
		}
	}()

	for i := start; i < end; i++ {
		_, _ = bar.Advance(1)
		time.Sleep(100 * time.Millisecond)
	}
}
