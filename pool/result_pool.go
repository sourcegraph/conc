package pool

import (
	"context"
	"sync"
)

func NewWithResults[T any]() *ResultPool[T] {
	return &ResultPool[T]{
		pool: *New(),
	}
}

type ResultPool[T any] struct {
	pool Pool
	agg  resultAggregator[T]
}

func (p *ResultPool[T]) Go(f func() T) {
	p.pool.Go(func() {
		p.agg.add(f())
	})
}

func (p *ResultPool[T]) Wait() []T {
	p.pool.Wait()
	return p.agg.results
}

func (p *ResultPool[T]) MaxGoroutines() int {
	return p.pool.MaxGoroutines()
}

func (p *ResultPool[T]) WithErrors() *ResultErrorPool[T] {
	return &ResultErrorPool[T]{
		errorPool: *p.pool.WithErrors(),
	}
}

func (p *ResultPool[T]) WithContext(ctx context.Context) *ResultContextPool[T] {
	return &ResultContextPool[T]{
		contextPool: *p.pool.WithContext(ctx),
	}
}

func (p *ResultPool[T]) WithMaxGoroutines(n int) *ResultPool[T] {
	p.pool.WithMaxGoroutines(n)
	return p
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
