package group

import (
	"context"
)

type ResultErrorGroup[T any] struct {
	errorGroup     ErrorGroup
	agg            resultAggregator[T]
	collectErrored bool
}

func (g *ResultErrorGroup[T]) Go(f func() (T, error)) {
	g.errorGroup.Go(func() error {
		res, err := f()
		if err == nil || g.collectErrored {
			g.agg.add(res)
		}
		return err
	})
}

func (g *ResultErrorGroup[T]) Wait() ([]T, error) {
	err := g.errorGroup.Wait()
	return g.agg.results, err
}

func (g *ResultErrorGroup[T]) WithCollectErrored() *ResultErrorGroup[T] {
	g.collectErrored = true
	return g
}

func (g *ResultErrorGroup[T]) WithFirstError() *ResultErrorGroup[T] {
	g.errorGroup.WithFirstError()
	return g
}

func (g *ResultErrorGroup[T]) WithMaxGoroutines(n int) *ResultErrorGroup[T] {
	g.errorGroup.WithMaxGoroutines(n)
	return g
}

func (g *ResultErrorGroup[T]) WithContext(ctx context.Context) *ResultContextGroup[T] {
	return &ResultContextGroup[T]{
		contextGroup: *g.errorGroup.WithContext(ctx),
	}
}
