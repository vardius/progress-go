package progress

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

const (
	_          = iota
	kb float64 = 1 << (10 * iota)
	mb
	gb
)

// Terminal colours.
const (
	clrReset  = "\x1b[0;1m"
	clrRed    = "\x1b[31;1m"
	clrGreen  = "\x1b[32;1m"
	clrYellow = "\x1b[33;1m"
	clrCyan   = "\x1b[36;1m"
)

type Bar struct {
	options *Options

	start   time.Time
	percent int64  // progress percentage
	step    int64  // current progress
	max     int64  // max value for progress
	rate    string // the actual progress bar to be printed
}

type Options struct {
	Output  io.Writer
	Graph   string
	Verbose bool
}

func New(start, max int64, opts ...Options) *Bar {
	opt := Options{
		Graph:  "█",
		Output: os.Stdout,
	}

	if len(opts) > 0 {
		if opts[0].Output != nil {
			opt.Output = opts[0].Output
		}
		if opts[0].Graph != "" {
			opt.Graph = opts[0].Graph
		}
		if opts[0].Verbose {
			opt.Verbose = true
		}
	}

	bar := Bar{
		options: &opt,
		step:    start,
		max:     max,
	}

	bar.percent = bar.getPercent()
	for i := 0; i < int(bar.percent); i += 2 {
		bar.rate += bar.options.Graph // initial progress position
	}

	return &bar
}

func (bar *Bar) Start() (n int, err error) {
	bar.start = time.Now()

	return bar.play(bar.step)
}

func (bar *Bar) Advance(inc int64) (n int, err error) {
	return bar.play(bar.step + inc)
}

func (bar *Bar) Stop() (n int, err error) {
	if n, err := bar.play(bar.max); err != nil {
		return n, err
	}

	return fmt.Fprintln(bar.options.Output)
}

func (bar *Bar) play(cur int64) (n int, err error) {
	last := bar.percent
	bar.step = cur
	bar.percent = bar.getPercent()

	for i := int64(0); i < (bar.percent-last)/2; i++ {
		bar.rate += bar.options.Graph
	}

	if !bar.options.Verbose {
		return fmt.Fprintf(bar.options.Output, "\r%8d/%d [%-50s] %3d%%", bar.step, bar.max, bar.rate, bar.percent)
	}

	memoryFormat, memory := bar.getMemory()

	return fmt.Fprintf(
		bar.options.Output,
		"\r%s %8d/%d "+clrGreen+"[%-50s]"+clrReset+" %3d%% "+memoryFormat+" %s/%s",
		bar.getRemaining(), bar.step, bar.max, bar.rate, bar.percent, memory, bar.getElapsed(), bar.getEstimated(),
	)
}

func (bar *Bar) getRate() int64 {
	if bar.max == 0 {
		return 50
	}
	// floor($this->max ? $this->percent * $this->barWidth : (null === $this->redrawFreq ? min(5, $this->barWidth / 15) * $this->writeCount : $this->step) % $this->barWidth)
	return int64(float32(bar.step) / float32(bar.max) * 100)
}

func (bar *Bar) getPercent() int64 {
	if bar.max == 0 {
		return 0
	}

	return int64(float32(bar.step) / float32(bar.max) * 100)
}

func (bar *Bar) getElapsed() time.Duration {
	return time.Since(bar.start).Round(time.Second)
}

func (bar *Bar) getEstimated() time.Duration {
	if bar.step == 0 {
		return 0
	}
	if bar.max == 0 {
		return 0
	}

	return (bar.getElapsed() * time.Duration(bar.max) / time.Duration(bar.step)).Round(time.Second)
}

func (bar *Bar) getRemaining() time.Duration {
	if bar.step == 0 {
		return 0
	}
	if bar.max == bar.step {
		return 0
	}

	return (bar.getElapsed() * time.Duration(bar.max-bar.step) / time.Duration(bar.step)).Round(time.Second)
}

func (bar *Bar) getMemory() (string, float64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memory := float64(m.TotalAlloc)

	if memory >= gb {
		return clrRed + "%.1f GB" + clrReset, memory / gb
	}
	if memory >= mb {
		return clrYellow + "%.1f MB" + clrReset, memory / mb
	}
	if memory >= kb {
		return clrCyan + "%.0f KB" + clrReset, memory / kb
	}

	return "%.0f B", memory
}
