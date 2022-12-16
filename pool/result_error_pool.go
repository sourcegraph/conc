package pool

import (
	"context"
)

type ResultErrorPool[T any] struct {
	errPool        ErrorPool
	agg            resultAggregator[T]
	collectErrored bool
}

func (p *ResultErrorPool[T]) Do(f func() (T, error)) {
	p.errPool.Go(func() error {
		res, err := f()
		if err == nil || p.collectErrored {
			p.agg.add(res)
		}
		return err
	})
}

func (p *ResultErrorPool[T]) Wait() ([]T, error) {
	err := p.errPool.Wait()
	return p.agg.results, err
}

func (p *ResultErrorPool[T]) WithCollectErrored() *ResultErrorPool[T] {
	p.collectErrored = true
	return p
}

func (p *ResultErrorPool[T]) WithFirstError() *ResultErrorPool[T] {
	p.errPool = *p.errPool.WithFirstError()
	return p
}

func (p *ResultErrorPool[T]) WithMaxGoroutines(limit int) *ResultErrorPool[T] {
	p.errPool = *p.errPool.WithMaxGoroutines(limit)
	return p
}

func (p *ResultErrorPool[T]) WithContext(ctx context.Context) *ResultContextPool[T] {
	return &ResultContextPool[T]{
		contextPool: *p.errPool.WithContext(ctx),
	}
}
