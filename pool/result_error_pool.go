package pool

import (
	"context"
)

type ResultErrorPool[T any] struct {
	ErrorPool
	agg            resultAggregator[T]
	collectErrored bool
}

func (p *ResultErrorPool[T]) Go(f func() (T, error)) {
	p.ErrorPool.Go(func() error {
		res, err := f()
		if err == nil || p.collectErrored {
			p.agg.add(res)
		}
		return err
	})
}

func (p *ResultErrorPool[T]) Wait() ([]T, error) {
	err := p.ErrorPool.Wait()
	return p.agg.results, err
}

func (p *ResultErrorPool[T]) WithCollectErrored() *ResultErrorPool[T] {
	p.collectErrored = true
	return p
}

func (p *ResultErrorPool[T]) WithContext(ctx context.Context) *ResultContextPool[T] {
	return &ResultContextPool[T]{
		ContextPool: *p.ErrorPool.WithContext(ctx),
	}
}
