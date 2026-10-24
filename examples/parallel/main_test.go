package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The SHA-256 of "abc", from FIPS 180-2.
const abc = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

func TestChecksum(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 20)
	for i := range paths {
		paths[i] = filepath.Join(dir, fmt.Sprint(i))
		if err := os.WriteFile(paths[i], []byte("abc"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	missing := filepath.Join(dir, "missing")
	paths[7] = missing

	var out bytes.Buffer
	sums, err := checksum(&out, paths, 4)

	if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), missing) {
		t.Errorf("checksum() error = %v, want %s not found", err, missing)
	}
	for i, s := range sums {
		want := abc
		if i == 7 {
			want = ""
		}
		if s != want {
			t.Errorf("sums[%d] = %q, want %q", i, s, want)
		}
	}
	// Every file counts as done, the missing one too.
	if want := "20/20 [" + strings.Repeat("█", 50) + "] 100%\n"; out.String() != want {
		t.Errorf("output %q, want %q", out.String(), want)
	}
}
