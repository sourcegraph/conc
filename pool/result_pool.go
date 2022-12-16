package pool

import (
	"context"
	"sync"
)

type ResultPool[T any] struct {
	Pool
	agg resultAggregator[T]
}

func (p *ResultPool[T]) Do(f func() T) {
	p.Pool.Go(func() {
		p.agg.add(f())
	})
}

func (p *ResultPool[T]) Wait() []T {
	p.Pool.Wait()
	return p.agg.results
}

func (p *ResultPool[T]) WithErrors() *ResultErrorPool[T] {
	return &ResultErrorPool[T]{
		ErrorPool: *p.Pool.WithErrors(),
	}
}

func (p *ResultPool[T]) WithContext(ctx context.Context) *ResultContextPool[T] {
	return &ResultContextPool[T]{
		ContextPool: *p.Pool.WithContext(ctx),
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
