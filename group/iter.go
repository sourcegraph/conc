package group

import (
	"github.com/camdencheek/conc"
)

func ForEachIdx[T any](input []T, f func(int, *T)) {
	conc.ForEachIdxIn(New(), input, f)
}

func ForEach[T any](input []T, f func(*T)) {
	conc.ForEachIdxIn(New(), input, func(_ int, t *T) {
		f(t)
	})
}

func Map[T any, R any](input []T, f func(*T) R) []R {
	return conc.MapIn(New(), input, f)
}

func MapErr[T any, R any](input []T, f func(*T) (R, error)) ([]R, error) {
	return conc.MapErrIn(New(), input, f)
}
