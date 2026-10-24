package main

import (
	"bytes"
	"regexp"
	"testing"
	"testing/synctest"
	"time"
)

func TestCount(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out bytes.Buffer
		if err := count(&out, 10, 100*time.Millisecond); err != nil {
			t.Fatalf("count() = %v", err)
		}

		// Not a terminal, so one plain line: the count, the bar, the heap size and 1s of 1s, none left.
		want := regexp.MustCompile(`^10/10 \[█{50}\] 100% [0-9.]+ (B|KB|MB|GB) 1s/1s 0s\n$`)
		if !want.MatchString(out.String()) {
			t.Errorf("output %q, want it to match %s", out.String(), want)
		}
	})
}
