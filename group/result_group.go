package group

import (
	"context"
	"sync"
)

type ResultGroup[T any] struct {
	Group
	agg resultAggregator[T]
}

func (g *ResultGroup[T]) Do(f func() T) {
	g.Group.Go(func() {
		g.agg.add(f())
	})
}

func (g *ResultGroup[T]) Wait() []T {
	g.Group.Wait()
	return g.agg.results
}

func (g *ResultGroup[T]) WithErrors() *ResultErrorGroup[T] {
	return &ResultErrorGroup[T]{
		ErrorGroup: *g.Group.WithErrors(),
	}
}

func (g *ResultGroup[T]) WithContext(ctx context.Context) *ResultContextGroup[T] {
	return &ResultContextGroup[T]{
		ContextGroup: *g.Group.WithContext(ctx),
	}
}

// resultAggregator is a utility type that lets us safely append from multiple
// goroutines. The zero value is valid and ready to use.
type resultAggregator[T any] struct {
	mu      sync.Mutex
	results []T
}

func (r *resultAggregator[T]) add(res T) {
	r.mu.Lock()
	r.results = append(r.results, res)
	r.mu.Unlock()
}
