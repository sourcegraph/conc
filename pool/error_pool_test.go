package pool

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func ExampleErrorPool() {
	p := New().WithErrors()
	for i := 0; i < 3; i++ {
		i := i
		p.Go(func() error {
			if i == 2 {
				return errors.New("oh no!")
			}
			return nil
		})
	}
	err := p.Wait()
	fmt.Println(err)
	// Output:
	// oh no!
}

func TestErrorPool(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")

	t.Run("wait returns no error if no errors", func(t *testing.T) {
		t.Parallel()
		g := New().WithErrors()
		g.Go(func() error { return nil })
		require.NoError(t, g.Wait())
	})

	t.Run("wait error if func returns error", func(t *testing.T) {
		t.Parallel()
		g := New().WithErrors()
		g.Go(func() error { return err1 })
		require.ErrorIs(t, g.Wait(), err1)
	})

	t.Run("wait error is all returned errors", func(t *testing.T) {
		t.Parallel()
		g := New().WithErrors()
		g.Go(func() error { return err1 })
		g.Go(func() error { return nil })
		g.Go(func() error { return err2 })
		err := g.Wait()
		require.ErrorIs(t, err, err1)
		require.ErrorIs(t, err, err2)
	})

	t.Run("propagates panics", func(t *testing.T) {
		t.Parallel()
		g := New().WithErrors()
		for i := 0; i < 10; i++ {
			i := i
			g.Go(func() error {
				if i == 5 {
					panic("fatal")
				}
				return nil
			})
		}
		require.Panics(t, func() { _ = g.Wait() })
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()
		for _, maxGoroutines := range []int{1, 10, 100} {
			t.Run(strconv.Itoa(maxGoroutines), func(t *testing.T) {
				g := New().WithErrors().WithMaxGoroutines(maxGoroutines)

				var currentConcurrent atomic.Int64
				taskCount := maxGoroutines * 10
				for i := 0; i < taskCount; i++ {
					g.Go(func() error {
						cur := currentConcurrent.Add(1)
						if cur > int64(maxGoroutines) {
							return errors.Newf("expected no more than %d concurrent goroutine", maxGoroutines)
						}
						time.Sleep(time.Millisecond)
						currentConcurrent.Add(-1)
						return nil
					})
				}
				require.NoError(t, g.Wait())
				require.Equal(t, int64(0), currentConcurrent.Load())
			})
		}
	})
}

func TestErrorPool_WithExitOnFirstError(t *testing.T) {
	var (
		p        = New().WithErrors().WithExitOnFirstError()
		runCount = atomic.Int32{} // should be 3
	)
	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)

		i := i
		p.Go(func() error {
			t.Log(i)
			runCount.Add(1)
			if i == 2 {
				return errors.New("demo error")
			}
			return nil
		})
	}
	_ = p.Wait()

	if c := runCount.Load(); c != 3 {
		t.Errorf("runCount should be 3, while got %d", c)
	}
}
