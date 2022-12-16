package pool

import (
	"runtime"

	"github.com/camdencheek/conc"
)

func New() Pool {
	return Pool{
		limiter: make(conc.Limiter, runtime.GOMAXPROCS(0)),
		tasks:   make(chan func()),
	}
}

type Pool struct {
	handle  conc.WaitGroup
	limiter conc.Limiter
	tasks   chan func()
}

func (p *Pool) Do(f func()) {
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

func (p Pool) WithMaxGoroutines(n int) Pool {
	p.limiter = make(chan struct{}, n)
	return p
}

func (p *Pool) MaxGoroutines() int {
	return cap(p.limiter)
}

func (p *Pool) worker() {
	for f := range p.tasks {
		f()
	}
}
