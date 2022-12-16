package group

import (
	"context"
)

type ResultContextGroup[T any] struct {
	contextGroup   ContextGroup
	agg            resultAggregator[T]
	collectErrored bool
}

func (g *ResultContextGroup[T]) Go(f func(context.Context) (T, error)) {
	g.contextGroup.Go(func(ctx context.Context) error {
		res, err := f(ctx)
		if err == nil || g.collectErrored {
			g.agg.add(res)
		}
		return err
	})
}

func (g *ResultContextGroup[T]) Wait() ([]T, error) {
	err := g.contextGroup.Wait()
	return g.agg.results, err
}

func (g *ResultContextGroup[T]) WithCollectErrored() *ResultContextGroup[T] {
	g.collectErrored = true
	return g
}

func (g *ResultContextGroup[T]) WithCancelOnError() *ResultContextGroup[T] {
	g.contextGroup.WithCancelOnError()
	return g
}

func (g *ResultContextGroup[T]) WithFirstError() *ResultContextGroup[T] {
	g.contextGroup.WithFirstError()
	return g
}

func (g *ResultContextGroup[T]) WithMaxGoroutines(n int) *ResultContextGroup[T] {
	g.contextGroup.WithMaxGoroutines(n)
	return g
}
