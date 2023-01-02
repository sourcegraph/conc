package pool

import (
	"context"
	"sync"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// ErrorPool is a pool that runs tasks that may return an error.
// Errors are collected and returned by Wait().
//
// A new ErrorPool should be created using `New().WithErrors()`.
type ErrorPool struct {
	pool Pool

	onlyFirstError bool

	mu   sync.Mutex
	errs error
}

// Go submits a task to the pool.
func (p *ErrorPool) Go(f func() error) {
	p.pool.Go(func() {
		p.addErr(f())
	})
}

// Wait cleans up any spawned goroutines, propagating any panics and
// returning any errors from tasks.
func (p *ErrorPool) Wait() error {
	p.pool.Wait()
	return p.errs
}

// WithContext converts the pool to a ContextPool for tasks that should
// be canceled on first error.
func (p *ErrorPool) WithContext(ctx context.Context) *ContextPool {
	ctx, cancel := context.WithCancel(ctx)
	return &ContextPool{
		errorPool: *p,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// WithFirstError configures the pool to only return the first error
// returned by a task. By default, Wait() will return a combined error.
func (p *ErrorPool) WithFirstError() *ErrorPool {
	p.onlyFirstError = true
	return p
}

// WithMaxGoroutines limits the number of goroutines in a pool.
// Defaults to runtime.GOMAXPROCS(0). Panics if n < 1.
func (p *ErrorPool) WithMaxGoroutines(n int) *ErrorPool {
	p.pool.WithMaxGoroutines(n)
	return p
}

func (p *ErrorPool) addErr(err error) {
	if err != nil {
		p.mu.Lock()
		if p.onlyFirstError {
			if p.errs == nil {
				p.errs = err
			}
		} else {
			p.errs = errors.Append(p.errs, err)
		}
		p.mu.Unlock()
	}
}
