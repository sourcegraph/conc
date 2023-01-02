package pool

import (
	"context"
)

type ResultContextPool[T any] struct {
	contextPool    ContextPool
	agg            resultAggregator[T]
	collectErrored bool
}

func (p *ResultContextPool[T]) Go(f func(context.Context) (T, error)) {
	p.contextPool.Go(func(ctx context.Context) error {
		res, err := f(ctx)
		if err == nil || p.collectErrored {
			p.agg.add(res)
		}
		return err
	})
}

func (p *ResultContextPool[T]) Wait() ([]T, error) {
	err := p.contextPool.Wait()
	return p.agg.results, err
}

func (p *ResultContextPool[T]) WithCollectErrored() *ResultContextPool[T] {
	p.collectErrored = true
	return p
}

func (p *ResultContextPool[T]) WithFirstError() *ResultContextPool[T] {
	p.contextPool.WithFirstError()
	return p
}

func (p *ResultContextPool[T]) WithMaxGoroutines(n int) *ResultContextPool[T] {
	p.contextPool.WithMaxGoroutines(n)
	return p
}
