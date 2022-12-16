package group

import (
	"github.com/camdencheek/conc"
)

func ForEachIdx[T any](input []T, f func(int, *T)) {
	g := New()
	defer g.Wait()

	conc.ForEachIdxIn(g, input, f)
}

func ForEach[T any](input []T, f func(*T)) {
	g := New()
	defer g.Wait()

	conc.ForEachIn(g, input, func(t *T) {
		f(t)
	})
}

func Map[T any, R any](input []T, f func(*T) R) []R {
	g := New()
	defer g.Wait()

	return conc.MapIn(g, input, f)
}

func MapErr[T any, R any](input []T, f func(*T) (R, error)) ([]R, error) {
	g := New()
	defer g.Wait()

	return conc.MapErrIn(New(), input, f)
}
