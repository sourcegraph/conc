package group

import (
	"context"
)

type ResultErrorGroup[T any] struct {
	errGroup       ErrorGroup
	agg            resultAggregator[T]
	collectErrored bool
}

func (g *ResultErrorGroup[T]) Do(f func() (T, error)) {
	g.errGroup.Go(func() error {
		res, err := f()
		if err == nil || g.collectErrored {
			g.agg.add(res)
		}
		return err
	})
}

func (g *ResultErrorGroup[T]) Wait() ([]T, error) {
	err := g.errGroup.Wait()
	return g.agg.results, err
}

func (g ResultErrorGroup[T]) WithCollectErrored() ResultErrorGroup[T] {
	g.collectErrored = true
	return g
}

func (g ResultErrorGroup[T]) WithFirstError() ResultErrorGroup[T] {
	g.errGroup = g.errGroup.WithFirstError()
	return g
}

func (g ResultErrorGroup[T]) WithMaxConcurrency(limit int) ResultErrorGroup[T] {
	g.errGroup = g.errGroup.WithMaxConcurrency(limit)
	return g
}

func (g ResultErrorGroup[T]) WithContext(ctx context.Context) ResultContextGroup[T] {
	return ResultContextGroup[T]{
		contextGroup: g.errGroup.WithContext(ctx),
	}
}
