package pool

import (
	"context"
)

type ResultContextPool[T any] struct {
	ContextPool
	agg            resultAggregator[T]
	collectErrored bool
}

func (p *ResultContextPool[T]) Do(f func(context.Context) (T, error)) {
	p.ContextPool.Do(func(ctx context.Context) error {
		res, err := f(ctx)
		if err == nil || p.collectErrored {
			p.agg.add(res)
		}
		return err
	})
}

func (p *ResultContextPool[T]) Wait() ([]T, error) {
	err := p.ContextPool.Wait()
	return p.agg.results, err
}

func (p *ResultContextPool[T]) WithCollectErrored() *ResultContextPool[T] {
	p.collectErrored = true
	return p
}
