package iter

import (
	"strconv"
	"testing"

	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIterator(t *testing.T) {
	t.Parallel()

	t.Run("safe for reuse", func(t *testing.T) {
		t.Parallel()

		iterator := Iterator[int]{Concurrency: 999}

		// iter.Concurrency > numInput case that updates iter.Concurrency
		iterator.ForEachIdx([]int{1, 2, 3}, func(i int, t *int) {})

		assert.Equal(t, iterator.Concurrency, 999)
	})
}

func TestForEachIdx(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{}
			ForEachIdx(ints, func(i int, val *int) {
				panic("this should never be called")
			})
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			ForEachIdx(ints, func(i int, val *int) {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		ForEachIdx(ints, func(i int, val *int) {
			*val += 1
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
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

func TestForEach(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{}
			ForEach(ints, func(val *int) {
				panic("this should never be called")
			})
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			ForEach(ints, func(val *int) {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		ForEach(ints, func(val *int) {
			*val += 1
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
		ints := make([]int, 10000)
		ForEach(ints, func(val *int) {
			*val = 1
		})
		expected := make([]int, 10000)
		for i := 0; i < 10000; i++ {
			expected[i] = 1
		}
		require.Equal(t, expected, ints)
	})
}

func TestMap(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{}
			Map(ints, func(val *int) int {
				panic("this should never be called")
			})
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			Map(ints, func(val *int) int {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine, though not recommended", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		Map(ints, func(val *int) int {
			*val += 1
			return 0
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("basic increment", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		res := Map(ints, func(val *int) int {
			return *val + 1
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, res)
		require.Equal(t, []int{1, 2, 3, 4, 5}, ints)
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
		ints := make([]int, 10000)
		res := Map(ints, func(val *int) int {
			return 1
		})
		expected := make([]int, 10000)
		for i := 0; i < 10000; i++ {
			expected[i] = 1
		}
		require.Equal(t, expected, res)
	})
}

func TestMapErr(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{}
			res, err := MapErr(ints, func(val *int) (int, error) {
				panic("this should never be called")
			})
			require.NoError(t, err)
			require.Equal(t, ints, res)
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			_, _ = MapErr(ints, func(val *int) (int, error) {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine, though not recommended", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		res, err := MapErr(ints, func(val *int) (int, error) {
			*val += 1
			return 0, nil
		})
		require.NoError(t, err)
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
		require.Equal(t, []int{0, 0, 0, 0, 0}, res)
	})

	t.Run("basic increment", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		res, err := MapErr(ints, func(val *int) (int, error) {
			return *val + 1, nil
		})
		require.NoError(t, err)
		require.Equal(t, []int{2, 3, 4, 5, 6}, res)
		require.Equal(t, []int{1, 2, 3, 4, 5}, ints)
	})

	err1 := errors.New("error1")
	err2 := errors.New("error1")

	t.Run("error is propagated", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		res, err := MapErr(ints, func(val *int) (int, error) {
			if *val == 3 {
				return 0, err1
			}
			return *val + 1, nil
		})
		require.ErrorIs(t, err, err1)
		require.Equal(t, []int{2, 3, 0, 5, 6}, res)
		require.Equal(t, []int{1, 2, 3, 4, 5}, ints)
	})

	t.Run("multiple errors are propagated", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		res, err := MapErr(ints, func(val *int) (int, error) {
			if *val == 3 {
				return 0, err1
			}
			if *val == 4 {
				return 0, err2
			}
			return *val + 1, nil
		})
		require.ErrorIs(t, err, err1)
		require.ErrorIs(t, err, err2)
		require.Equal(t, []int{2, 3, 0, 0, 6}, res)
		require.Equal(t, []int{1, 2, 3, 4, 5}, ints)
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
		ints := make([]int, 10000)
		res := Map(ints, func(val *int) int {
			return 1
		})
		expected := make([]int, 10000)
		for i := 0; i < 10000; i++ {
			expected[i] = 1
		}
		require.Equal(t, expected, res)
	})
}

func BenchmarkForEach(b *testing.B) {
	for _, count := range []int{0, 1, 8, 100, 1000, 10000, 100000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			ints := make([]int, count)
			for i := 0; i < b.N; i++ {
				ForEach(ints, func(i *int) {
					*i = 0
				})
			}
		})
	}
}
