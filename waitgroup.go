package conc

import (
	"sync"
)

// WaitGroup is the primary building block for scoped concurrency.
// Goroutines can be spawned in the WaitGroup with the Go method,
// and calling Wait() will ensure that each of those goroutines exits
// before continuing. Any panics in a child goroutine will be caught
// and propagated to the caller of Wait().
type WaitGroup struct {
	wg sync.WaitGroup
	pc PanicCatcher
}

// Go spawns a new goroutine in the WaitGroup
func (h *WaitGroup) Go(f func()) {
	h.wg.Add(1)
	go func() {
		defer h.done()
		h.pc.Try(f)
	}()
}

// Wait will block until all goroutines spawned with Go exit and will
// propagate any panics spawned in a child goroutine.
func (h *WaitGroup) Wait() {
	h.wg.Wait()

	// Propagate a panic if we caught one from a child goroutine
	if r := h.pc.Value(); r != nil {
		panic(r.Value)
	}
}

// done should be called in a defer statement in a child goroutine.
// It catches any panics and decrements the counter.
func (h *WaitGroup) done() {
	h.wg.Done()
}
