package pool

import (
	"context"
	"runtime"

	"github.com/camdencheek/conc"
)

func New() *Pool {
	return &Pool{
		limiter: make(conc.Limiter, runtime.GOMAXPROCS(0)),
		// tasks is not buffered because if it were, it would be possible to
		// submit tasks but never be forced to start a worker goroutine. If
		// goroutines were started eagerly, this wouldn't be a problem, but
		// I think it's preferable to minimize the cost of creating a pool.
		tasks: make(chan func()),
	}
}

type Pool struct {
	handle  conc.WaitGroup
	limiter conc.Limiter
	tasks   chan func()
}

func (p *Pool) Go(f func()) {
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
	close(p.tasks)
	p.handle.Wait()
}

func (p *Pool) MaxGoroutines() int {
	return p.limiter.Limit()
}

func (p *Pool) WithMaxGoroutines(n int) *Pool {
	p.limiter = make(chan struct{}, n)
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
