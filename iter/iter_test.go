package iter

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExampleIterator() {
	input := []int{1, 2, 3, 4}
	iterator := Iterator[int]{
		MaxGoroutines: len(input) / 2,
	}

	iterator.ForEach(input, func(v *int) {
		if *v%2 != 0 {
			*v = -1
		}
	})

	fmt.Println(input)
	// Output:
	// [-1 2 -1 4]
}

func TestIterator(t *testing.T) {
	t.Parallel()

	t.Run("safe for reuse", func(t *testing.T) {
		t.Parallel()

		iterator := Iterator[int]{MaxGoroutines: 999}

		// iter.Concurrency > numInput case that updates iter.Concurrency
		iterator.ForEachIdx([]int{1, 2, 3}, func(i int, t *int) {})

		require.Equal(t, iterator.MaxGoroutines, 999)
	})

	t.Run("allows more than defaultMaxGoroutines() concurrent tasks", func(t *testing.T) {
		t.Parallel()

		wantConcurrency := 2 * defaultMaxGoroutines()

		maxConcurrencyHit := make(chan struct{})

		tasks := make([]int, wantConcurrency)
		iterator := Iterator[int]{MaxGoroutines: wantConcurrency}

		var concurrentTasks atomic.Int64
		iterator.ForEach(tasks, func(t *int) {
			n := concurrentTasks.Add(1)
			defer concurrentTasks.Add(-1)

			if int(n) == wantConcurrency {
				// All our tasks are running concurrently.
				// Signal to the rest of the tasks to stop.
				close(maxConcurrencyHit)
			} else {
				// Wait until we hit max concurrency before exiting.
				// This ensures that all tasks have been started
				// in parallel, despite being a larger input set than
				// defaultMaxGoroutines().
				<-maxConcurrencyHit
			}
		})
	})
}

func TestForEachIdxCtx(t *testing.T) {
	t.Parallel()

	bgctx := context.Background()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{}
			err := ForEachIdxCtx(bgctx, ints, func(ctx context.Context, i int, val *int) error {
				panic("this should never be called")
			})
			require.NoError(t, err)
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			_ = ForEachIdxCtx(bgctx, ints,
				func(ctx context.Context, i int, val *int) error {
					panic("super bad thing happened")
				})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		err := ForEachIdxCtx(bgctx, ints, func(ctx context.Context, i int, val *int) error {
			*val += 1
			return nil
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
		require.NoError(t, err)
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
		ints := make([]int, 10000)
		err := ForEachIdxCtx(bgctx, ints, func(ctx context.Context, i int, val *int) error {
			*val = i
			return nil
		})
		expected := make([]int, 10000)
		for i := 0; i < 10000; i++ {
			expected[i] = i
		}
		require.Equal(t, expected, ints)
		require.NoError(t, err)
	})

	err1 := errors.New("error1")
	err2 := errors.New("error2")

	t.Run("first error is propagated", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		err := ForEachIdxCtx(bgctx, ints, func(ctx context.Context, i int, val *int) error {
			if *val == 3 {
				return err1
			}
			if *val == 4 {
				return err2
			}
			return nil
		})
		require.ErrorIs(t, err, err1)
		require.NotErrorIs(t, err, err2)
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

func TestForEachCtx(t *testing.T) {
	t.Parallel()

	bgctx := context.Background()
	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		err := ForEachCtx(bgctx, ints, func(ctx context.Context, val *int) error {
			*val += 1
			return nil
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
		require.NoError(t, err)
	})

	err1 := errors.New("error1")
	err2 := errors.New("error2")

	t.Run("first error is propagated", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		err := ForEachCtx(bgctx, ints, func(ctx context.Context, val *int) error {
			if *val == 3 {
				return err1
			}
			if *val == 4 {
				return err2
			}
			return nil
		})
		require.ErrorIs(t, err, err1)
		require.NotErrorIs(t, err, err2)
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
