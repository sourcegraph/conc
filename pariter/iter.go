package pariter

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/camdencheek/conc"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type GoWaiter interface {
	Go(func())
	Wait()
}

func ForEachIdx[T any](input []T, f func(int, *T)) {
	var wg conc.WaitGroup
	defer wg.Wait()

	ForEachIdxIn(&wg, input, f)
}

func ForEach[T any](input []T, f func(*T)) {
	var wg conc.WaitGroup
	defer wg.Wait()

	ForEachIdxIn(&wg, input, func(_ int, t *T) {
		f(t)
	})
}

func Map[T any, R any](input []T, f func(*T) R) []R {
	var wg conc.WaitGroup
	defer wg.Wait()

	return MapIn(&wg, input, f)
}

func MapErr[T any, R any](input []T, f func(*T) (R, error)) ([]R, error) {
	var wg conc.WaitGroup
	defer wg.Wait()

	return MapErrIn(&wg, input, f)
}

func ForEachIdxIn[T any](goer GoWaiter, input []T, f func(int, *T)) {
	numTasks := runtime.GOMAXPROCS(0)
	if m, ok := goer.(interface{ MaxGoroutines() int }); ok && m.MaxGoroutines() > numTasks {
		// No more tasks than the pool's max concurrency
		numTasks = m.MaxGoroutines()
	}
	if numTasks > len(input) {
		// No more tasks than the number of input items
		numTasks = len(input)
	}

	var idx atomic.Int64
	var wg sync.WaitGroup
	task := func() {
		defer wg.Done()
		i := int(idx.Add(1) - 1)
		for ; i < len(input); i = int(idx.Add(1) - 1) {
			f(i, &input[i])
		}
	}

	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		goer.Go(task)
	}
	wg.Wait()
}

func ForEachIn[T any](goer GoWaiter, input []T, f func(*T)) {
	ForEachIdxIn(goer, input, func(_ int, t *T) {
		f(t)
	})
}

func MapIn[T any, R any](goer GoWaiter, input []T, f func(*T) R) []R {
	res := make([]R, len(input))
	ForEachIdxIn(goer, input, func(i int, t *T) {
		res[i] = f(t)
	})
	return res
}

func MapErrIn[T any, R any](goer GoWaiter, input []T, f func(*T) (R, error)) ([]R, error) {
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
