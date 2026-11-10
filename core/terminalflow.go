package core

import (
	"io"
	"sync"
)

const terminalOutputWindow = 1024 * 1024

// Browser acknowledgements stop a fast remote command from growing xterm's
// parsing queue indefinitely. Input and resize messages use a separate reader.
type terminalFlow struct {
	mu      sync.Mutex
	cond    *sync.Cond
	pending int
	closed  bool
}

func newTerminalFlow() *terminalFlow {
	f := &terminalFlow{}
	f.cond = sync.NewCond(&f.mu)
	return f
}

func (f *terminalFlow) reserve(n int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for !f.closed && f.pending+n > terminalOutputWindow {
		f.cond.Wait()
	}
	if f.closed {
		return io.ErrClosedPipe
	}
	f.pending += n
	return nil
}

func (f *terminalFlow) acknowledge(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n > 0 && n <= f.pending {
		f.pending -= n
		f.cond.Broadcast()
	}
}

func (f *terminalFlow) close() {
	f.mu.Lock()
	f.closed = true
	f.cond.Broadcast()
	f.mu.Unlock()
}
