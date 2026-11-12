package conc

import (
	"iter"
	"sync"

	"github.com/sourcegraph/conc/panics"
)

// NewWaitGroup creates a new WaitGroup.
func NewWaitGroup() *WaitGroup {
	return &WaitGroup{}
}

// WaitGroup is the primary building block for scoped concurrency.
// Goroutines can be spawned in the WaitGroup with the Go method,
// and calling Wait() will ensure that each of those goroutines exits
// before continuing. Any panics in a child goroutine will be caught
// and propagated to the caller of Wait().
//
// The zero value of WaitGroup is usable, just like sync.WaitGroup.
// Also like sync.WaitGroup, it must not be copied after first use.
type WaitGroup struct {
	wg sync.WaitGroup
	pc panics.Catcher
}

// Go spawns a new goroutine in the WaitGroup.
func (h *WaitGroup) Go(f func()) {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.pc.Try(f)
	}()
}

// GoForEach executes the given function concurrently for each element in the iterator.
// It maintains the order of the input iterator in the execution. This method is useful
// for parallel processing of iterable data structures while preserving their original sequence.
func (h *WaitGroup) GoForEach(seq iter.Seq[func()]) {
	for f := range seq {
		h.Go(f)
	}
}

// Wait will block until all goroutines spawned with Go exit and will
// propagate any panics spawned in a child goroutine.
func (h *WaitGroup) Wait() {
	h.wg.Wait()

	// Propagate a panic if we caught one from a child goroutine.
	h.pc.Repanic()
}

// WaitAndRecover will block until all goroutines spawned with Go exit and
// will return a *panics.Recovered if one of the child goroutines panics.
func (h *WaitGroup) WaitAndRecover() *panics.Recovered {
	h.wg.Wait()

	// Return a recovered panic if we caught one from a child goroutine.
	return h.pc.Recovered()
}
