package pool

import (
	"sync"
	"sync/atomic"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func ForEachIdx[T any](input []T, f func(int, *T)) {
	p := New()
	ForEachIdxIn(&p, input, f)
}

func ForEachIdxIn[T any](p *Pool, input []T, f func(int, *T)) {
	var idx atomic.Int64
	var wg sync.WaitGroup
	task := func() {
		defer wg.Done()
		for i := int(idx.Add(1) - 1); i < len(input); i = int(idx.Add(1) - 1) {
			f(i, &input[i])
		}
	}

	n := p.MaxGoroutines()
	wg.Add(n)
	for i := 0; i < p.MaxGoroutines(); i++ {
		p.Do(task)
	}

	wg.Wait()
}

func ForEach[T any](input []T, f func(*T)) {
	p := New()
	ForEachIdxIn(&p, input, func(_ int, t *T) {
		f(t)
	})
}

func ForEachIn[T any](p *Pool, input []T, f func(*T)) {
	ForEachIdxIn(p, input, func(_ int, t *T) {
		f(t)
	})
}

func Map[T any, R any](input []T, f func(*T) R) []R {
	p := New()
	return MapIn(&p, input, f)
}

func MapIn[T any, R any](p *Pool, input []T, f func(*T) R) []R {
	res := make([]R, len(input))
	ForEachIdxIn(p, input, func(i int, t *T) {
		res[i] = f(t)
	})
	return res
}

func MapErr[T any, R any](input []T, f func(*T) (R, error)) ([]R, error) {
	p := New()
	return MapErrIn(&p, input, f)
}

func MapErrIn[T any, R any](p *Pool, input []T, f func(*T) (R, error)) ([]R, error) {
	var (
		res    = make([]R, len(input))
		errMux sync.Mutex
		errs   error
	)
	ForEachIdxIn(p, input, func(i int, t *T) {
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
