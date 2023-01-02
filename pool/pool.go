package pool

import (
	"context"
	"runtime"
	"sync"

	"github.com/camdencheek/conc"
)

// New creates a new Pool.
func New() *Pool {
	return &Pool{}
}

// Pool is a pool of goroutines used to execute tasks concurrently.
//
// Tasks are submitted with Go(). Once all your tasks have been submitted, you
// must call Wait() to clean up any spawned goroutines and propagate any
// panics.
//
// Goroutines are started lazily, so creating a new pool is cheap. There will
// never be more goroutines spawned than there are tasks submitted.
//
// Pool is efficient, but not zero cost. It should not be used for very short
// tasks. Startup and teardown come with an overhead of around 1µs, and each
// task has an overhead of around 300ns.
type Pool struct {
	handle   conc.WaitGroup
	limiter  limiter
	tasks    chan func()
	initOnce sync.Once
}

// Go submits a task to be run in the pool.
func (p *Pool) Go(f func()) {
	p.init()

	select {
	case p.limiter <- struct{}{}:
		// If we are below our limit, spawn a new worker rather
		// than waiting for one to become available.
		p.handle.Go(p.worker)

		// We know there is a least one worker running, so wait
		// for it to become available. This ensures we never spawn
		// more workers than the number of tasks.
		p.tasks <- f
	case p.tasks <- f:
		// A worker is available and has accepted the task
		return
	}
}

// Wait cleans up spawned goroutines, propagating any panics that were
// raised by a tasks.
func (p *Pool) Wait() {
	p.init()

	close(p.tasks)
	p.handle.Wait()
}

// MaxGoroutines returns the maximum size of the pool.
func (p *Pool) MaxGoroutines() int {
	return p.limiter.limit()
}

// WithMaxGoroutines limits the number of goroutines in a pool.
// Defaults to runtime.GOMAXPROCS(0). Panics if n < 1.
func (p *Pool) WithMaxGoroutines(n int) *Pool {
	if n < 1 {
		panic("max goroutines in a pool must be greater than zero")
	}
	p.limiter = make(limiter, n)
	return p
}

// init ensures that the pool is initialized before use. This makes the
// zero value of the pool usable.
func (p *Pool) init() {
	p.initOnce.Do(func() {
		// Do not override the limiter if set by WithMaxGoroutines
		if p.limiter == nil {
			p.limiter = make(limiter, runtime.GOMAXPROCS(0))
		}

		p.tasks = make(chan func())
	})
}

// WithErrors converts the pool to an ErrorPool so the submitted tasks can
// return errors.
func (p *Pool) WithErrors() *ErrorPool {
	return &ErrorPool{
		pool: *p,
	}
}

// WithContext converts the pool to a ContextPool for tasks that should
// be canceled on first error.
func (p *Pool) WithContext(ctx context.Context) *ContextPool {
	ctx, cancel := context.WithCancel(ctx)
	return &ContextPool{
		errorPool: *p.WithErrors(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (p *Pool) worker() {
	// The only time this matters is if the task panics.
	// This makes it possible to spin up new workers in that case.
	defer p.limiter.release()

	for f := range p.tasks {
		f()
	}
}

type limiter chan struct{}

func (l limiter) limit() int {
	return cap(l)
}

func (l limiter) acquire() {
	l <- struct{}{}
}

func (l limiter) release() {
	<-l
}
