package iter

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/camdencheek/conc"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// ForEach executes f in parallel over each element in input.
//
// It is safe to mutate the input parameter, which makes it
// possible to map in place.
//
// ForEach always uses at most runtime.GOMAXPROCS goroutines.
// It takes roughly 2µs to start up the goroutines and adds
// an overhead of roughly 50ns per element of input.
func ForEach[T any](input []T, f func(*T)) {
	ForEachIdx(input, func(_ int, t *T) {
		f(t)
	})
}

// ForEachIdx is the same as ForEach except it also provides the
// index of the element to the callback.
func ForEachIdx[T any](input []T, f func(int, *T)) {
	var wg conc.WaitGroup
	defer wg.Wait()

	numTasks := runtime.GOMAXPROCS(0)
	if numTasks > len(input) {
		// No more tasks than the number of input items
		numTasks = len(input)
	}

	var idx atomic.Int64
	// create the task outside the loop to avoid extra allocations
	task := func() {
		i := int(idx.Add(1) - 1)
		for ; i < len(input); i = int(idx.Add(1) - 1) {
			f(i, &input[i])
		}
	}

	for i := 0; i < numTasks; i++ {
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
func MapErr[T any, R any](input []T, f func(*T) (R, error)) ([]R, error) {
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
			errs = errors.Append(errs, err)
			errMux.Unlock()
		}
	})
	return res, errs
}
