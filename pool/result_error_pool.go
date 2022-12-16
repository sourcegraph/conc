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
	p.errPool.Do(func() error {
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

func (g ResultErrorPool[T]) WithCollectErrored() ResultErrorPool[T] {
	g.collectErrored = true
	return g
}

func (g ResultErrorPool[T]) WithFirstError() ResultErrorPool[T] {
	g.errPool = g.errPool.WithFirstError()
	return g
}

func (g ResultErrorPool[T]) WithMaxGoroutines(limit int) ResultErrorPool[T] {
	g.errPool = g.errPool.WithMaxGoroutines(limit)
	return g
}

func (g ResultErrorPool[T]) WithContext(ctx context.Context) ResultContextPool[T] {
	return ResultContextPool[T]{
		contextPool: g.errPool.WithContext(ctx),
	}
}
