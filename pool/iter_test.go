package pool

import (
	"strconv"
	"testing"

	"github.com/camdencheek/conc"
)

func BenchmarkForEachIdx(b *testing.B) {
	b.Run("simple mutation", func(b *testing.B) {
		for _, n := range []int{10, 1000, 1000000} {
			arr := make([]int, n)
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				b.Run("baseline", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						for j := 0; j < len(arr); j++ {
							arr[j] = j
						}
					}
				})

				b.Run("parallel", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						ForEachIdx(arr, func(i int, val *int) {
							*val = i
						})
					}
				})

				g := New()
				b.Run("share group", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						conc.ForEachIdxIn(g, arr, func(i int, val *int) {
							*val = i
						})
					}
				})
			})
		}
	})
}

func BenchmarkForEach(b *testing.B) {
	b.Run("simple mutation", func(b *testing.B) {
		for _, n := range []int{10, 1000, 1000000} {
			arr := make([]int, n)
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				b.Run("baseline", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						for j := 0; j < len(arr); j++ {
							arr[j] = j
						}
					}
				})

				b.Run("parallel", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						ForEach(arr, func(val *int) {
							*val = i
						})
					}
				})

				g := New()
				b.Run("share group", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						conc.ForEachIn(g, arr, func(val *int) {
							*val = i
						})
					}
				})
			})
		}
	})
}
