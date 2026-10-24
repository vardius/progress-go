package progress

// Seams at the system's edges: whether Output is a terminal, and the runtime's heap size.

// IsTerminal is isTerminal, for tests.
var IsTerminal = isTerminal

// NewTerminal is New with Output taken to be a terminal, which otherwise only an *os.File can be.
func NewTerminal(total int64, opts Options) *Bar {
	b := New(total, opts)
	b.tty = true
	return b
}

// SetHeap makes a verbose bar show n bytes as the heap size.
func (b *Bar) SetHeap(n uint64) {
	b.heap = func() (uint64, bool) { return n, true }
}

// Draw draws one frame, as a redraw does.
func (b *Bar) Draw() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.draw("")
}
