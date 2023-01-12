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

	onlyFirstError   bool
	exitOnFirstError bool

	mu   sync.RWMutex
	errs error
}

// Go submits a task to the pool.
func (p *ErrorPool) Go(f func() error) {
	if p.shouldExitOnFirstError() {
		return
	}

	p.pool.Go(func() {
		if p.shouldExitOnFirstError() {
			return
		}

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
		errorPool: p.deref(),
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

// WithExitOnFirstError configures the pool to exit on the first error
// encountered. By default, Wait() will always run Go().
func (p *ErrorPool) WithExitOnFirstError() *ErrorPool {
	p.exitOnFirstError = true
	return p
}

// WithMaxGoroutines limits the number of goroutines in a pool.
// Defaults to unlimited. Panics if n < 1.
func (p *ErrorPool) WithMaxGoroutines(n int) *ErrorPool {
	p.pool.WithMaxGoroutines(n)
	return p
}

// deref is a helper that creates a shallow copy of the pool with the same
// settings. We don't want to just dereference the pointer because that makes
// the copylock lint angry.
func (p *ErrorPool) deref() ErrorPool {
	return ErrorPool{
		pool:           p.pool.deref(),
		onlyFirstError: p.onlyFirstError,
	}
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

func (p *ErrorPool) shouldExitOnFirstError() bool {
	if !p.exitOnFirstError {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errs != nil
}
