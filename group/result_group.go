package group

import (
	"context"
	"sync"
)

func NewWithResults[T any]() *ResultGroup[T] {
	return &ResultGroup[T]{}
}

type ResultGroup[T any] struct {
	group Group
	agg   resultAggregator[T]
}

func (g *ResultGroup[T]) Go(f func() T) {
	g.group.Go(func() {
		g.agg.add(f())
	})
}

func (g *ResultGroup[T]) Wait() []T {
	g.group.Wait()
	return g.agg.results
}

func (g *ResultGroup[T]) MaxGoroutines() int {
	return g.group.MaxGoroutines()
}

func (g *ResultGroup[T]) WithMaxGoroutines(n int) *ResultGroup[T] {
	g.group.WithMaxGoroutines(n)
	return g
}

func (g *ResultGroup[T]) WithErrors() *ResultErrorGroup[T] {
	return &ResultErrorGroup[T]{
		errorGroup: *g.group.WithErrors(),
	}
}

func (g *ResultGroup[T]) WithContext(ctx context.Context) *ResultContextGroup[T] {
	return &ResultContextGroup[T]{
		contextGroup: *g.group.WithContext(ctx),
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
