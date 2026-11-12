package iter

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/sourcegraph/conc"
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

	// FailFast controls whether the ForEach*Err methods should
	// stop iterating over the input if an error is returned by callback.
	FailFast bool
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

// ForEachIdxErr is the same as ForEachIdx except it returns an error.
// This function will not stop iterating over the input if an error is returned by callback.
// All returned errors will be returned as a multierror.
func ForEachIdxErr[T any](input []T, f func(int, *T) error) error {
	return Iterator[T]{}.ForEachIdxErr(input, f)
}

// ForEachErr is the same as ForEach except it returns an error.
// This function will not stop iterating over the input if an error is returned by callback.
func ForEachErr[T any](input []T, f func(*T) error) error {
	return Iterator[T]{}.ForEachErr(input, f)
}

// ForEachIdx is the same as ForEach except it also provides the
// index of the element to the callback.
func (iter Iterator[T]) ForEachIdx(input []T, f func(int, *T)) {
	if iter.MaxGoroutines == 0 {
		// iter is a value receiver and is hence safe to mutate
		iter.MaxGoroutines = defaultMaxGoroutines()
	}

	numInput := len(input)
	if iter.MaxGoroutines > numInput {
		// No more concurrent tasks than the number of input items.
		iter.MaxGoroutines = numInput
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
	for i := 0; i < iter.MaxGoroutines; i++ {
		wg.Go(task)
	}
	wg.Wait()
}

// ForEachErr is the same as ForEach except it returns an error.
// If Iterator.FailFast is true, ForEachErr will stop iterating on the first error.
// If Iterator.FailFast is false, ForEachErr will return all errors and iterate over the all the input.
func (iter Iterator[T]) ForEachErr(input []T, f func(*T) error) error {
	return iter.ForEachIdxErr(input, func(_ int, t *T) error {
		return f(t)
	})
}

// ForEachIdxErr is the same as ForEachIdx except it returns an error.
// If Iterator.FailFast is true, ForEachIdxErr will stop iterating on the first error.
// If Iterator.FailFast is false, ForEachIdxErr will return all errors and iterate over the all the input.
func (iter Iterator[T]) ForEachIdxErr(input []T, f func(int, *T) error) error {
	if iter.MaxGoroutines == 0 {
		// iter is a value receiver and is hence safe to mutate
		iter.MaxGoroutines = defaultMaxGoroutines()
	}

	numInput := len(input)
	if iter.MaxGoroutines > numInput {
		// No more concurrent tasks than the number of input items.
		iter.MaxGoroutines = numInput
	}

	var errs []error
	var errsMu sync.Mutex
	var idx atomic.Int64
	var failed atomic.Bool

	// Create the task outside the loop to avoid extra closure allocations.
	task := func() {
		i := int(idx.Add(1) - 1)
		for ; i < numInput && !failed.Load(); i = int(idx.Add(1) - 1) {
			if err := f(i, &input[i]); err != nil {
				if alreadyFailedFast := failed.Swap(iter.FailFast); !alreadyFailedFast {
					errsMu.Lock()
					errs = append(errs, err)
					errsMu.Unlock()
				}

				failed.Store(iter.FailFast)
			}
		}
	}

	var wg conc.WaitGroup
	for i := 0; i < iter.MaxGoroutines && !failed.Load(); i++ {
		wg.Go(task)
	}
	wg.Wait()

	return errors.Join(errs...)
}
