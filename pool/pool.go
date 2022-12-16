package pool

import (
	"context"
	"runtime"

	"github.com/camdencheek/conc"
)

func New() *Pool {
	return &Pool{
		limiter: make(conc.Limiter, runtime.GOMAXPROCS(0)),
		tasks:   make(chan func(), 8),
	}
}

type Pool struct {
	handle  conc.WaitGroup
	limiter conc.Limiter
	tasks   chan func()
}

func (p *Pool) Go(f func()) {
	for {
		select {
		case p.limiter <- struct{}{}:
			p.handle.Go(p.worker)
		case p.tasks <- f:
			return
		}
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

func (p Pool) WithContext(ctx context.Context) *ContextPool {
	return &ContextPool{
		ErrorPool: *p.WithErrors(),
		ctx:       ctx,
	}
}

func (p *Pool) worker() {
	for f := range p.tasks {
		f()
	}
}
