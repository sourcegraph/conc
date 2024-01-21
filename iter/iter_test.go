package iter_test

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sourcegraph/conc/iter"

	"github.com/stretchr/testify/require"
)

func ExampleIterator() {
	input := []int{1, 2, 3, 4}
	iterator := iter.Iterator[int]{
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

		iterator := iter.Iterator[int]{MaxGoroutines: 999}

		// iter.Concurrency > numInput case that updates iter.Concurrency
		iterator.ForEachIdx([]int{1, 2, 3}, func(i int, t *int) {})

		require.Equal(t, iterator.MaxGoroutines, 999)
	})

	t.Run("allows more than defaultMaxGoroutines() concurrent tasks", func(t *testing.T) {
		t.Parallel()

		wantConcurrency := 2 * iter.DefaultMaxGoroutines()

		maxConcurrencyHit := make(chan struct{})

		tasks := make([]int, wantConcurrency)
		iterator := iter.Iterator[int]{MaxGoroutines: wantConcurrency}

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

func TestForEachIdx(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{}
			iter.ForEachIdx(ints, func(i int, val *int) {
				panic("this should never be called")
			})
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			iter.ForEachIdx(ints, func(i int, val *int) {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		iter.ForEachIdx(ints, func(i int, val *int) {
			*val += 1
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
		ints := make([]int, 10000)
		iter.ForEachIdx(ints, func(i int, val *int) {
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
			iter.ForEach(ints, func(val *int) {
				panic("this should never be called")
			})
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			iter.ForEach(ints, func(val *int) {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		iter.ForEach(ints, func(val *int) {
			*val += 1
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
		ints := make([]int, 10000)
		iter.ForEach(ints, func(val *int) {
			*val = 1
		})
		expected := make([]int, 10000)
		for i := 0; i < 10000; i++ {
			expected[i] = 1
		}
		require.Equal(t, expected, ints)
	})
}

func TestForIterator_EachIdxErr(t *testing.T) {
	t.Parallel()

	t.Run("failFast=false", func(t *testing.T) {
		it := Iterator[int]{MaxGoroutines: 999}
		forEach := noIndex(it.ForEachIdxErr)
		testForEachErr(t, false, forEach)
	})

	t.Run("failFast=true", func(t *testing.T) {
		it := Iterator[int]{MaxGoroutines: 999}
		forEach := noIndex(it.ForEachIdxErr)
		testForEachErr(t, true, forEach)
	})

	t.Run("failfast", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5}
		errTest := errors.New("test error")
		iterator := Iterator[int]{MaxGoroutines: 1, FailFast: true}

		var mu sync.Mutex
		var results []int

		err := iterator.ForEachIdxErr(input, func(_ int, t *int) error {
			mu.Lock()
			results = append(results, *t)
			mu.Unlock()

			return errTest
		})

		require.ErrorIs(t, err, errTest)
		require.Len(t, results, 1, "results")
		require.Containsf(t, input, results[0], "results")
	})

	t.Run("safe for reuse", func(t *testing.T) {
		t.Parallel()

		iterator := Iterator[int]{MaxGoroutines: 999}

		// iter.Concurrency > numInput case that updates iter.Concurrency
		_ = iterator.ForEachIdxErr([]int{1, 2, 3}, func(i int, t *int) error {
			return nil
		})

		require.Equal(t, iterator.MaxGoroutines, 999)
	})

	t.Run("allows more than defaultMaxGoroutines() concurrent tasks", func(t *testing.T) {
		t.Parallel()

		wantConcurrency := 2 * defaultMaxGoroutines()

		maxConcurrencyHit := make(chan struct{})

		tasks := make([]int, wantConcurrency)
		iterator := Iterator[int]{MaxGoroutines: wantConcurrency}

		var concurrentTasks atomic.Int64
		_ = iterator.ForEachIdxErr(tasks, func(_ int, t *int) error {
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

			return nil
		})
	})
}

func TestForIterator_EachErr(t *testing.T) {
	t.Parallel()

	t.Run("failFast=false", func(t *testing.T) {
		it := Iterator[int]{MaxGoroutines: 999}
		testForEachErr(t, false, it.ForEachErr)
	})

	t.Run("failFast=true", func(t *testing.T) {
		it := Iterator[int]{MaxGoroutines: 999}
		testForEachErr(t, true, it.ForEachErr)
	})

	t.Run("safe for reuse", func(t *testing.T) {
		t.Parallel()

		iterator := Iterator[int]{MaxGoroutines: 999}

		// iter.Concurrency > numInput case that updates iter.Concurrency
		_ = iterator.ForEachErr([]int{1, 2, 3}, func(t *int) error {
			return nil
		})

		require.Equal(t, iterator.MaxGoroutines, 999)
	})

	t.Run("failfast", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5}
		errTest := errors.New("test error")
		iterator := Iterator[int]{MaxGoroutines: 1, FailFast: true}

		var mu sync.Mutex
		var results []int

		err := iterator.ForEachErr(input, func(t *int) error {
			mu.Lock()
			results = append(results, *t)
			mu.Unlock()

			return errTest
		})

		require.ErrorIs(t, err, errTest)
		require.Len(t, results, 1, "results")
		require.Containsf(t, input, results[0], "results")
	})

	t.Run("allows more than defaultMaxGoroutines() concurrent tasks", func(t *testing.T) {
		t.Parallel()

		wantConcurrency := 2 * defaultMaxGoroutines()

		maxConcurrencyHit := make(chan struct{})

		tasks := make([]int, wantConcurrency)
		iterator := Iterator[int]{MaxGoroutines: wantConcurrency}

		var concurrentTasks atomic.Int64
		_ = iterator.ForEachErr(tasks, func(t *int) error {
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

			return nil
		})
	})
}

func TestForEachIdxErr(t *testing.T) {
	t.Parallel()

	t.Run("standart", func(t *testing.T) {
		forEach := noIndex(ForEachIdxErr[int])
		testForEachErr(t, false, forEach)
	})

	t.Run("unique indexes", func(t *testing.T) {
		ints := []int{1, 2, 3, 4, 5}
		got := []int{}
		gotMu := sync.Mutex{}

		err := ForEachIdxErr(ints, func(i int, _ *int) error {
			gotMu.Lock()
			defer gotMu.Unlock()
			got = append(got, i)
			return nil
		})

		require.NoError(t, err)
		require.ElementsMatch(t, got, []int{0, 1, 2, 3, 4})
	})
}

func TestForEachErr(t *testing.T) {
	t.Parallel()

	testForEachErr(t, false, ForEachErr[int])
}

// noIndex converts a ForEachIdxErr function (or method) into a ForEachErr function (or method).
func noIndex(forEach func([]int, func(int, *int) error) error) func([]int, func(*int) error) error {
	return func(ints []int, fn func(*int) error) error {
		return forEach(ints, func(_ int, val *int) error {
			return fn(val)
		})
	}
}

// testForEachErr runs a set of tests against a ForEachErr function (or method).
func testForEachErr(t *testing.T, failFast bool, forEach func([]int, func(*int) error) error) {
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{}
			_ = forEach(ints, func(val *int) error {
				panic("this should never be called")
			})
		}
		require.NotPanics(t, f)
	})

	t.Run("panic is propagated", func(t *testing.T) {
		t.Parallel()
		f := func() {
			ints := []int{1}
			_ = forEach(ints, func(val *int) error {
				panic("super bad thing happened")
			})
		}
		require.Panics(t, f)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		_ = forEach(ints, func(val *int) error {
			*val += 1
			return nil
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("mutating inputs is fine", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		_ = forEach(ints, func(val *int) error {
			*val += 1
			return nil
		})
		require.Equal(t, []int{2, 3, 4, 5, 6}, ints)
	})

	t.Run("returning errors", func(t *testing.T) {
		t.Parallel()
		ints := []int{1, 2, 3, 4, 5}
		errs := []error{
			errors.New("error 1"),
			errors.New("error 2"),
		}

		err := forEach(ints, func(i *int) error {
			return errs[*i%len(errs)]
		})

		switch {
		case failFast:
			assertAnyErrorIs(t, err, errs)
		default:
			assertAllErrorIs(t, err, errs)
		}
	})

	t.Run("huge inputs", func(t *testing.T) {
		t.Parallel()
		ints := make([]int, 10000)
		_ = forEach(ints, func(val *int) error {
			*val = 1

			return nil
		})
		expected := make([]int, 10000)
		for i := 0; i < 10000; i++ {
			expected[i] = 1
		}
		require.Equal(t, expected, ints)
	})
}

func assertAllErrorIs(t *testing.T, errs error, targets []error) {
	t.Helper()

	for _, target := range targets {
		if !errors.Is(errs, target) {
			t.Errorf("expected error to be %v, got %v", target, errs)
		}
	}
}

func assertAnyErrorIs(t *testing.T, errs error, targets []error) {
	t.Helper()

	for _, target := range targets {
		if errors.Is(errs, target) {
			return
		}
	}

	t.Errorf("expected any error to be one of %v, got %v", targets, errs)
}

func BenchmarkForEach(b *testing.B) {
	for _, count := range []int{0, 1, 8, 100, 1000, 10000, 100000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			ints := make([]int, count)
			for i := 0; i < b.N; i++ {
				iter.ForEach(ints, func(i *int) {
					*i = 0
				})
			}
		})
	}
}
