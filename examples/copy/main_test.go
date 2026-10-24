package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	data := bytes.Repeat([]byte("progress"), 1<<17) // 1 MiB, copied in several writes
	if err := os.WriteFile(src, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := copyFile(&out, dst, src); err != nil {
		t.Fatalf("copyFile() = %v", err)
	}

	if got, err := os.ReadFile(dst); err != nil || !bytes.Equal(got, data) {
		t.Errorf("dst holds %d bytes, err %v; want the %d bytes of src", len(got), err, len(data))
	}
	if want := "1048576/1048576 [" + strings.Repeat("█", 50) + "] 100%\n"; out.String() != want {
		t.Errorf("output %q, want %q", out.String(), want)
	}
}

func TestCopyFileMissing(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	err := copyFile(&out, filepath.Join(dir, "dst"), filepath.Join(dir, "missing"))

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("copyFile() = %v, want %v", err, fs.ErrNotExist)
	}
	if out.Len() != 0 {
		t.Errorf("output %q, want none: the bar never started", out.String())
	}
}
