package group

import (
	"context"
)

type ResultErrorGroup[T any] struct {
	ErrorGroup
	agg            resultAggregator[T]
	collectErrored bool
}

func (g *ResultErrorGroup[T]) Do(f func() (T, error)) {
	g.ErrorGroup.Go(func() error {
		res, err := f()
		if err == nil || g.collectErrored {
			g.agg.add(res)
		}
		return err
	})
}

func (g *ResultErrorGroup[T]) Wait() ([]T, error) {
	err := g.ErrorGroup.Wait()
	return g.agg.results, err
}

func (g *ResultErrorGroup[T]) WithCollectErrored() *ResultErrorGroup[T] {
	g.collectErrored = true
	return g
}

func (g *ResultErrorGroup[T]) WithContext(ctx context.Context) *ResultContextGroup[T] {
	return &ResultContextGroup[T]{
		ContextGroup: *g.ErrorGroup.WithContext(ctx),
	}
}
