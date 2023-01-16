package iter

import (
	"sync"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

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
