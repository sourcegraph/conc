package pool

import (
	"github.com/camdencheek/conc"
)

func ForEachIdx[T any](input []T, f func(int, *T)) {
	p := New()
	defer p.Wait()

	conc.ForEachIdxIn(p, input, f)
}

func ForEach[T any](input []T, f func(*T)) {
	p := New()
	defer p.Wait()

	conc.ForEachIdxIn(p, input, func(_ int, t *T) {
		f(t)
	})
}

func Map[T any, R any](input []T, f func(*T) R) []R {
	p := New()
	defer p.Wait()

	return conc.MapIn(p, input, f)
}

func MapErr[T any, R any](input []T, f func(*T) (R, error)) ([]R, error) {
	p := New()
	defer p.Wait()

	return conc.MapErrIn(p, input, f)
}
