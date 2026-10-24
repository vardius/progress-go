//go:build race

package progress_test

// raceEnabled is true when built with -race, which changes allocation counts.
const raceEnabled = true
