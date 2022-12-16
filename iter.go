package conc

import (
	"sync"
	"sync/atomic"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type Goer interface {
	Go(func())
	MaxGoroutines() int
	Wait()
}

func ForEachIdxIn[T any](goer Goer, input []T, f func(int, *T)) {
	var idx atomic.Int64
	var wg sync.WaitGroup
	task := func() {
		defer wg.Done()
		for i := int(idx.Add(1) - 1); i < len(input); i = int(idx.Add(1) - 1) {
			f(i, &input[i])
		}
	}

	n := goer.MaxGoroutines()
	wg.Add(n)
	for i := 0; i < n; i++ {
		goer.Go(task)
	}

	wg.Wait()
}

func ForEachIn[T any](goer Goer, input []T, f func(*T)) {
	ForEachIdxIn(goer, input, func(_ int, t *T) {
		f(t)
	})
}

func MapIn[T any, R any](goer Goer, input []T, f func(*T) R) []R {
	res := make([]R, len(input))
	ForEachIdxIn(goer, input, func(i int, t *T) {
		res[i] = f(t)
	})
	return res
}

func MapErrIn[T any, R any](goer Goer, input []T, f func(*T) (R, error)) ([]R, error) {
	var (
		res    = make([]R, len(input))
		errMux sync.Mutex
		errs   error
	)
	ForEachIdxIn(goer, input, func(i int, t *T) {
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
