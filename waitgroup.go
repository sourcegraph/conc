package conc

import (
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// WaitGroup is the primary building block for scoped concurrency.
// Goroutines can be spawned in the WaitGroup with the Go method,
// and calling Wait() will ensure that each of those goroutines exits
// before continuing. Any panics in a child goroutine will be caught
// and propagated to the caller of Wait().
type WaitGroup struct {
	wg          sync.WaitGroup
	caughtPanic atomic.Pointer[error]
}

// Go spawns a new goroutine in the WaitGroup
func (h *WaitGroup) Go(f func()) {
	h.wg.Add(1)
	go func() {
		defer h.done()
		f()
	}()
}

// Wait will block until all goroutines spawned with Go exit and will
// propagate any panics spawned in a child goroutine.
func (h *WaitGroup) Wait() {
	h.wg.Wait()

	// Propagate a panic if we caught one from a child goroutine
	if r := h.caughtPanic.Load(); r != nil {
		panic(*r)
	}
}

// done should be called in a defer statement in a child goroutine
func (h *WaitGroup) done() {
	if r := recover(); r != nil {
		err := errors.Newf(
			"recovered from panic in child goroutine: %#v\n\nchild stacktrace:\n%s\n",
			r,
			debug.Stack(),
		)
		h.caughtPanic.Store(&err)
	}
	h.wg.Done()
}
