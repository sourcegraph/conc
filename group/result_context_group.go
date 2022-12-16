package group

import (
	"context"
)

type ResultContextGroup[T any] struct {
	ContextGroup
	agg            resultAggregator[T]
	collectErrored bool
}

func (g *ResultContextGroup[T]) Go(f func(context.Context) (T, error)) {
	g.ContextGroup.Go(func(ctx context.Context) error {
		res, err := f(ctx)
		if err == nil || g.collectErrored {
			g.agg.add(res)
		}
		return err
	})
}

func (g *ResultContextGroup[T]) Wait() ([]T, error) {
	err := g.ContextGroup.Wait()
	return g.agg.results, err
}

func (g *ResultContextGroup[T]) WithCollectErrored() *ResultContextGroup[T] {
	g.collectErrored = true
	return g
}
