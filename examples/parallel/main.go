// Command parallel checksums files on four goroutines that share one bar.
//
//	go run ./examples/parallel <file>...
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/vardius/progress-go"
)

// checksum returns the SHA-256 of each file, hashed by up to workers goroutines at once.
// Each advances the bar when it finishes a file.
func checksum(out io.Writer, paths []string, workers int) ([]string, error) {
	sums := make([]string, len(paths))
	errs := make([]error, len(paths))
	bar := progress.New(int64(len(paths)), progress.Options{Output: out})
	bar.Start()

	next := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for i := range next {
				sums[i], errs[i] = sum(paths[i])
				bar.Advance(1)
			}
		})
	}
	for i := range paths {
		next <- i
	}
	close(next)
	wg.Wait()

	return sums, errors.Join(append(errs, bar.Stop())...)
}

func sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func main() {
	log.SetFlags(0)
	sums, err := checksum(os.Stderr, os.Args[1:], 4)
	for i, s := range sums {
		if s != "" {
			fmt.Printf("%s  %s\n", s, os.Args[i+1])
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
