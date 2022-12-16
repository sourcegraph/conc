package pool

import (
	"context"
)

type ResultErrorPool[T any] struct {
	errorPool      ErrorPool
	agg            resultAggregator[T]
	collectErrored bool
}

func (p *ResultErrorPool[T]) Go(f func() (T, error)) {
	p.errorPool.Go(func() error {
		res, err := f()
		if err == nil || p.collectErrored {
			p.agg.add(res)
		}
		return err
	})
}

func (p *ResultErrorPool[T]) Wait() ([]T, error) {
	err := p.errorPool.Wait()
	return p.agg.results, err
}

func (p *ResultErrorPool[T]) WithCollectErrored() *ResultErrorPool[T] {
	p.collectErrored = true
	return p
}

func (p *ResultErrorPool[T]) WithContext(ctx context.Context) *ResultContextPool[T] {
	return &ResultContextPool[T]{
		contextPool: *p.errorPool.WithContext(ctx),
	}
}

func (p *ResultErrorPool[T]) WithFirstError() *ResultErrorPool[T] {
	p.errorPool.WithFirstError()
	return p
}

func (p *ResultErrorPool[T]) WithMaxGoroutines(n int) *ResultErrorPool[T] {
	p.errorPool.WithMaxGoroutines(n)
	return p
}
