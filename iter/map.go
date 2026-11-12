package iter

import (
	"context"
	"errors"
	"sync"
)

// Mapper is an Iterator with a result type R. It can be used to configure
// the behaviour of Map and MapErr. The zero value is safe to use with
// reasonable defaults.
//
// Mapper is also safe for reuse and concurrent use.
type Mapper[T, R any] Iterator[T]

// Map applies f to each element of input, returning the mapped result.
//
// Map always uses at most runtime.GOMAXPROCS goroutines. For a configurable
// goroutine limit, use a custom Mapper.
func Map[T, R any](input []T, f func(*T) R) []R {
	return Mapper[T, R]{}.Map(input, f)
}

// Map applies f to each element of input, returning the mapped result.
//
// Map uses up to the configured Mapper's maximum number of goroutines.
func (m Mapper[T, R]) Map(input []T, f func(*T) R) []R {
	res, _ := m.MapCtx(context.Background(), input, func(_ context.Context, t *T) (R, error) {
		return f(t), nil
	})
	return res
}

// MapErr applies f to each element of the input, returning the mapped result
// and a combined error of all returned errors.
//
// Map always uses at most runtime.GOMAXPROCS goroutines. For a configurable
// goroutine limit, use a custom Mapper.
func MapErr[T, R any](input []T, f func(*T) (R, error)) ([]R, error) {
	return Mapper[T, R]{}.MapErr(input, f)
}

// MapErr applies f to each element of the input, returning the mapped result
// and a combined error of all returned errors.
//
// Map uses up to the configured Mapper's maximum number of goroutines.
func (m Mapper[T, R]) MapErr(input []T, f func(*T) (R, error)) ([]R, error) {
	var (
		errMux sync.Mutex
		errs   []error
	)
	// MapErr handles its own errors by accumulating them as a multierror, ignoring the error from MapCtx which is only the first error
	res, _ := m.MapCtx(context.Background(), input, func(ctx context.Context, t *T) (R, error) {
		ires, err := f(t)
		if err != nil {
			errMux.Lock()
			errs = append(errs, err)
			errMux.Unlock()
		}
		return ires, nil
	})
	return res, errors.Join(errs...)
}

// MapCtx is the same as Map except it also accepts a context
// that it uses to manages the execution of tasks.
// The context is cancelled on task failure and the first error is returned.
func MapCtx[T, R any](ctx context.Context, input []T, f func(context.Context, *T) (R, error)) ([]R, error) {
	return Mapper[T, R]{}.MapCtx(ctx, input, f)
}

// MapCtx is the same as Map except it also accepts a context
// that it uses to manages the execution of tasks.
// The context is cancelled on task failure and the first error is returned.
func (m Mapper[T, R]) MapCtx(ctx context.Context, input []T, f func(context.Context, *T) (R, error)) ([]R, error) {
	var (
		res = make([]R, len(input))
	)
	return res, Iterator[T](m).ForEachIdxCtx(ctx, input, func(innerctx context.Context, i int, t *T) error {
		var err error
		res[i], err = f(innerctx, t)
		return err
	})
}
