// Command copy copies a file under a bar that counts the bytes.
//
//	go run ./examples/copy <src> <dst>
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/vardius/progress-go"
)

// copyFile copies src to dst. The bar's total is src's size, and it advances by every byte written.
func copyFile(out io.Writer, dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	to, err := os.Create(dst)
	if err != nil {
		return err
	}

	bar := progress.New(info.Size(), progress.Options{Output: out})
	bar.Start()
	_, err = io.Copy(io.MultiWriter(to, counter{bar}), in)
	if err != nil {
		err = fmt.Errorf("copy %s to %s: %w", src, dst, err)
	}
	return errors.Join(err, to.Close(), bar.Stop())
}

// counter is a writer that advances a bar by the length of every write.
type counter struct{ bar *progress.Bar }

func (c counter) Write(p []byte) (int, error) {
	c.bar.Advance(int64(len(p)))
	return len(p), nil
}

func main() {
	log.SetFlags(0)
	if len(os.Args) != 3 {
		log.Fatal("usage: copy <src> <dst>")
	}
	if err := copyFile(os.Stdout, os.Args[2], os.Args[1]); err != nil {
		log.Fatal(err)
	}
}
