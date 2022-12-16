package pool

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/camdencheek/conc"

	"github.com/sourcegraph/sourcegraph/lib/errors"
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

func ForEachIdx[T any](p *Pool, input []T, f func(int, *T)) {
	var idx atomic.Int64
	var wg sync.WaitGroup
	task := func() {
		defer wg.Done()
		for i := int(idx.Add(1) - 1); i < len(input); i = int(idx.Add(1) - 1) {
			f(i, &input[i])
		}
	}

	n := p.MaxGoroutines()
	wg.Add(n)
	for i := 0; i < p.MaxGoroutines(); i++ {
		p.Do(task)
	}

	wg.Wait()
}

func ForEach[T any](p *Pool, input []T, f func(*T)) {
	ForEachIdx(p, input, func(_ int, t *T) {
		f(t)
	})
}

func Map[T any, R any](p *Pool, input []T, f func(*T) R) []R {
	res := make([]R, len(input))
	ForEachIdx(p, input, func(i int, t *T) {
		res[i] = f(t)
	})
	return res
}

func MapErr[T any, R any](p *Pool, input []T, f func(*T) (R, error)) ([]R, error) {
	var (
		res    = make([]R, len(input))
		errMux sync.Mutex
		errs   error
	)
	ForEachIdx(p, input, func(i int, t *T) {
		var err error
		res[i], err = f(t)
		if err != nil {
			errMux.Lock()
			errs = errors.Append(errs, err)
			errMux.Unlock()
		}
	})
	return res, errs
}
