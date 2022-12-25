package pool

import (
	"context"
	"runtime"
	"sync"

	"github.com/camdencheek/conc"
)

func New() *Pool {
	return &Pool{}
}

type Pool struct {
	handle   conc.WaitGroup
	limiter  conc.Limiter
	tasks    chan func()
	initOnce sync.Once
}

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

func (p *Pool) Wait() {
	p.init()
	close(p.tasks)
	p.handle.Wait()
}

func (p *Pool) MaxGoroutines() int {
	return p.limiter.Limit()
}

// init ensures that the pool is initialized before use. This makes the
// zero value of the pool usable.
func (p *Pool) init() {
	p.initOnce.Do(func() {
		// Do not override the limiter if set by WithMaxGoroutines
		if p.limiter == nil {
			p.limiter = make(conc.Limiter, runtime.GOMAXPROCS(0))
		}

		p.tasks = make(chan func())
	})
}

// WithMaxGoroutines limits the number of goroutines in a pool.
// Panics if n < 1.
func (p *Pool) WithMaxGoroutines(n int) *Pool {
	if n < 1 {
		panic("max goroutines in a pool must be greater than zero")
	}
	p.limiter = conc.NewLimiter(n)
	return p
}

func (p *Pool) WithErrors() *ErrorPool {
	return &ErrorPool{
		pool: *p,
	}
}

func (p *Pool) WithContext(ctx context.Context) *ContextPool {
	return &ContextPool{
		errorPool: *p.WithErrors(),
		ctx:       ctx,
	}
}

func (p *Pool) worker() {
	// The only time this matters is if the task panics.
	// This makes it possible to spin up new workers in that case.
	defer p.limiter.Release()

	for f := range p.tasks {
		f()
	}
}
