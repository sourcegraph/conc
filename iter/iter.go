package iter

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/sourcegraph/conc"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// Iterator can be used to control iterations like ForEach.
// The zero value is safe to use with reasonable defaults.
type Iterator[T any] struct {
	// Concurrency, if unset, defaults to runtime.GOMAXPROCS(0).
	Concurrency int
}

// ForEach executes f in parallel over each element in input.
//
// It is safe to mutate the input parameter, which makes it
// possible to map in place.
//
// ForEach always uses at most runtime.GOMAXPROCS goroutines.
// It takes roughly 2µs to start up the goroutines and adds
// an overhead of roughly 50ns per element of input.
func ForEach[T any](input []T, f func(*T)) { Iterator[T]{}.ForEach(input, f) }

func (iter Iterator[T]) ForEach(input []T, f func(*T)) {
	iter.ForEachIdx(input, func(_ int, t *T) {
		f(t)
	})
}

// ForEachIdx is the same as ForEach except it also provides the
// index of the element to the callback.
func ForEachIdx[T any](input []T, f func(int, *T)) { Iterator[T]{}.ForEachIdx(input, f) }

func (iter Iterator[T]) ForEachIdx(input []T, f func(int, *T)) {
	if iter.Concurrency == 0 {
		iter.Concurrency = runtime.GOMAXPROCS(0)
	}

	numInput := len(input)
	if iter.Concurrency > numInput {
		// No more concurrent tasks than the number of input items.
		iter.Concurrency = numInput
	}

	var idx atomic.Int64
	// Create the task outside the loop to avoid extra closure allocations.
	task := func() {
		i := int(idx.Add(1) - 1)
		for ; i < numInput; i = int(idx.Add(1) - 1) {
			f(i, &input[i])
		}
	}

	var wg conc.WaitGroup
	for i := 0; i < iter.Concurrency; i++ {
		wg.Go(task)
	}
	wg.Wait()
}

// Map applies f to each element of input, returning the mapped result.
func Map[T, R any](input []T, f func(*T) R) []R {
	res := make([]R, len(input))
	ForEachIdx(input, func(i int, t *T) {
		res[i] = f(t)
	})
	return res
}

// MapErr applies f to each element of the input, returning the mapped result
// and a combined error of all returned errors.
func MapErr[T, R any](input []T, f func(*T) (R, error)) ([]R, error) {
	var (
		res    = make([]R, len(input))
		errMux sync.Mutex
		errs   error
	)
	ForEachIdx(input, func(i int, t *T) {
		var err error
		res[i], err = f(t)
		if err != nil {
			errMux.Lock()
			// TODO: use stdlib errors once multierrors land in go 1.20
			errs = errors.Append(errs, err)
			errMux.Unlock()
		}
	})
	return res, errs
}
