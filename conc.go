package conc

import (
	"sync"
	"sync/atomic"
)

type WaitGroup struct {
	wg          sync.WaitGroup
	caughtPanic atomic.Pointer[any]
}

func (h *WaitGroup) Spawn(f func()) {
	h.wg.Add(1)
	go func() {
		defer h.done()
		f()
	}()
}

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
		h.caughtPanic.Store(&r)
	}
	h.wg.Done()
}
