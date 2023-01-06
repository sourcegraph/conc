package conc

import (
	"context"
	"sync"
	"time"
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
		defer h.wg.Done()
		h.pc.Try(f)
	}()
}

// GoWithContext spawns a new goroutine in the WaitGroup
func (h *WaitGroup) GoWithContext(f func(), ctx context.Context) {
	h.wg.Add(1)
	result := make(chan struct{}, 1)
	defer h.wg.Done()

	go func() {
		result <- h.pc.TryWithReturn(f)
	}()

	select {
	case <-ctx.Done():
		return
	case _ = <-result:
		return
	}
}

// GoWithTimeout spawns a new goroutine in the WaitGroup
func (h *WaitGroup) GoWithTimeout(f func(), timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	h.GoWithContext(f, ctx)
}

// Wait will block until all goroutines spawned with Go exit and will
// propagate any panics spawned in a child goroutine.
func (h *WaitGroup) Wait() {
	h.wg.Wait()

	// Propagate a panic if we caught one from a child goroutine
	h.pc.Repanic()
}
