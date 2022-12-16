package group

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func TestErrorGroup(t *testing.T) {
	t.Parallel()
	err1 := errors.New("err1")
	err2 := errors.New("err2")

	t.Run("wait returns no error if no errors", func(t *testing.T) {
		g := New().WithErrors()
		g.Go(func() error { return nil })
		require.NoError(t, g.Wait())
	})

	t.Run("wait error if func returns error", func(t *testing.T) {
		g := New().WithErrors()
		g.Go(func() error { return err1 })
		require.ErrorIs(t, g.Wait(), err1)
	})

	t.Run("wait error is all returned errors", func(t *testing.T) {
		g := New().WithErrors()
		g.Go(func() error { return err1 })
		g.Go(func() error { return nil })
		g.Go(func() error { return err2 })
		require.ErrorIs(t, g.Wait(), err1)
		require.ErrorIs(t, g.Wait(), err2)
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
