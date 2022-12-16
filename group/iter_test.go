package group

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/camdencheek/conc"
)

func TestForEachIdx(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		f := func() {
			ints := []int{}
			ForEachIdx(ints, func(i int, val *int) {
				panic("this should never be called")
			})
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		f := func() {
			ints := []int{1}
			ForEachIdx(ints, func(i int, val *int) {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		ints := []int{1, 2, 3, 4, 5}
		ForEachIdx(ints, func(i int, val *int) {
			*val += 1
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("huge inputs", func(t *testing.T) {
		ints := make([]int, 10000)
		ForEachIdx(ints, func(i int, val *int) {
			*val = i
		})
		expected := make([]int, 10000)
		for i := 0; i < 10000; i++ {
			expected[i] = i
		}
		require.Equal(t, expected, ints)
	})
}

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
