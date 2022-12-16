package group

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGroup(t *testing.T) {
	t.Parallel()
	t.Run("basic", func(t *testing.T) {
		g := New()
		var completed atomic.Int64
		for i := 0; i < 100; i++ {
			g.Go(func() {
				time.Sleep(10 * time.Millisecond)
				completed.Add(1)
			})
		}
		g.Wait()
		require.Equal(t, completed.Load(), int64(100))
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()
		for _, maxConcurrent := range []int{1, 10, 100} {
			t.Run(strconv.Itoa(maxConcurrent), func(t *testing.T) {
				g := New().WithMaxConcurrency(maxConcurrent)

				currentConcurrent := atomic.NewInt64(0)
				errCount := atomic.NewInt64(0)
				taskCount := maxConcurrent * 10
				for i := 0; i < taskCount; i++ {
					g.Go(func() {
						cur := currentConcurrent.Inc()
						if cur > int64(maxConcurrent) {
							errCount.Inc()
						}
						time.Sleep(time.Millisecond)
						currentConcurrent.Dec()
					})
				}
				g.Wait()
				require.Equal(t, int64(0), errCount.Load())
				require.Equal(t, int64(0), currentConcurrent.Load())
			})
		}
	})

	t.Run("propagate panic", func(t *testing.T) {
		g := New()
		for i := 0; i < 10; i++ {
			i := i
			g.Go(func() {
				if i == 5 {
					panic(i)
				}
			})
		}
		require.Panics(t, func() { g.Wait() })
	})
}
