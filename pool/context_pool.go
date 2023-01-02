package pool

import (
	"context"
)

// ContextPool is a pool that runs tasks that take a context.
// The context passed to the task will be canceled if any
// of the tasks return an error, which makes its functionality
// different than just capturing a context with the task closure.
//
// A new ContextPool should be created with `New().WithContext(ctx)`.
type ContextPool struct {
	errorPool ErrorPool

	ctx    context.Context
	cancel context.CancelFunc
}

// Go submits a task. If it returns an error, the error will be
// collected and returned by Wait() and the context passed to other
// tasks will be canceled.
func (g *ContextPool) Go(f func(ctx context.Context) error) {
	g.errorPool.Go(func() error {
		err := f(g.ctx)
		if err != nil {
			// Leaky abstraction warning: We add the error directly because
			// otherwise, canceling could cause another goroutine to exit and
			// return an error before this error was added, which breaks the
			// expectations of WithFirstError().
			g.errorPool.addErr(err)
			g.cancel()
			return nil
		}
		return err
	})
}

// Wait cleans up all spawned goroutines, propagates any panics, and
// returns an error if any of the tasks errored.
func (p *ContextPool) Wait() error {
	return p.errorPool.Wait()
}

// WithFirstError configures the pool to only return the first error
// returned by a task. By default, Wait() will return a combined error.
// This is particularly useful for ContextPool where all errors after the
// first are likely to be context.Canceled.
func (p *ContextPool) WithFirstError() *ContextPool {
	p.errorPool.WithFirstError()
	return p
}

// WithMaxGoroutines limits the number of goroutines in a pool.
// Defaults to runtime.GOMAXPROCS(0). Panics if n < 1.
func (p *ContextPool) WithMaxGoroutines(n int) *ContextPool {
	p.errorPool.WithMaxGoroutines(n)
	return p
}
