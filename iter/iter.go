package iter

import (
	"context"
	"runtime"
	"sync/atomic"

	"github.com/sourcegraph/conc/pool"
)

// defaultMaxGoroutines returns the default maximum number of
// goroutines to use within this package.
func defaultMaxGoroutines() int { return runtime.GOMAXPROCS(0) }

// Iterator can be used to configure the behaviour of ForEach
// and ForEachIdx. The zero value is safe to use with reasonable
// defaults.
//
// Iterator is also safe for reuse and concurrent use.
type Iterator[T any] struct {
	// MaxGoroutines controls the maximum number of goroutines
	// to use on this Iterator's methods.
	//
	// If unset, MaxGoroutines defaults to runtime.GOMAXPROCS(0).
	MaxGoroutines int
}

// ForEach executes f in parallel over each element in input.
//
// It is safe to mutate the input parameter, which makes it
// possible to map in place.
//
// ForEach always uses at most runtime.GOMAXPROCS goroutines.
// It takes roughly 2µs to start up the goroutines and adds
// an overhead of roughly 50ns per element of input. For
// a configurable goroutine limit, use a custom Iterator.
func ForEach[T any](input []T, f func(*T)) { Iterator[T]{}.ForEach(input, f) }

// ForEach executes f in parallel over each element in input,
// using up to the Iterator's configured maximum number of
// goroutines.
//
// It is safe to mutate the input parameter, which makes it
// possible to map in place.
//
// It takes roughly 2µs to start up the goroutines and adds
// an overhead of roughly 50ns per element of input.
func (iter Iterator[T]) ForEach(input []T, f func(*T)) {
	iter.ForEachIdx(input, func(_ int, t *T) {
		f(t)
	})
}

// ForEachIdx is the same as ForEach except it also provides the
// index of the element to the callback.
func ForEachIdx[T any](input []T, f func(int, *T)) { Iterator[T]{}.ForEachIdx(input, f) }

// ForEachIdx is the same as ForEach except it also provides the
// index of the element to the callback.
func (iter Iterator[T]) ForEachIdx(input []T, f func(int, *T)) {
	_ = iter.ForEachIdxCtx(context.Background(), input, func(_ context.Context, idx int, input *T) error {
		f(idx, input)
		return nil
	})
}

func ForEachIdxCtx[T any](octx context.Context, input []T, f func(context.Context, int, *T) error) error {
	return Iterator[T]{}.ForEachIdxCtx(octx, input, f)
}

func ForEachCtx[T any](octx context.Context, input []T, f func(context.Context, *T) error) error {
	return Iterator[T]{}.ForEachCtx(octx, input, f)
}

func (iter Iterator[T]) ForEachCtx(octx context.Context, input []T, f func(context.Context, *T) error) error {
	return iter.ForEachIdxCtx(octx, input, func(ctx context.Context, _ int, input *T) error {
		return f(ctx, input)
	})
}

func (iter Iterator[T]) ForEachIdxCtx(octx context.Context, input []T, f func(context.Context, int, *T) error) error {
	if iter.MaxGoroutines == 0 {
		// iter is a value receiver and is hence safe to mutate
		iter.MaxGoroutines = defaultMaxGoroutines()
	}

	numInput := len(input)
	if iter.MaxGoroutines > numInput && numInput > 0 {
		// No more concurrent tasks than the number of input items.
		iter.MaxGoroutines = numInput
	}

	var idx atomic.Int64
	// Create the task outside the loop to avoid extra closure allocations.
	task := func(ctx context.Context) error {
		i := int(idx.Add(1) - 1)
		for ; i < numInput && ctx.Err() == nil; i = int(idx.Add(1) - 1) {
			if err := f(ctx, i, &input[i]); err != nil {
				return err
			}
		}
		return nil
	}

	runner := pool.New().
		WithContext(octx).
		WithCancelOnError().
		WithFirstError().
		WithMaxGoroutines(iter.MaxGoroutines)
	for i := 0; i < iter.MaxGoroutines; i++ {
		runner.Go(task)
	}
	return runner.Wait()
}
