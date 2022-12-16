package group

import (
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResultGroup(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		g := NewWithResults[int]()
		expected := []int{}
		for i := 0; i < 100; i++ {
			i := i
			expected = append(expected, i)
			g.Go(func() int {
				return i
			})
		}
		res := g.Wait()
		sort.Ints(res)
		require.Equal(t, expected, res)
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()
		for _, maxGoroutines := range []int{1, 10, 100} {
			t.Run(strconv.Itoa(maxGoroutines), func(t *testing.T) {
				g := NewWithResults[int]().WithMaxGoroutines(maxGoroutines)

				var currentConcurrent atomic.Int64
				var errCount atomic.Int64
				taskCount := maxGoroutines * 10
				expected := make([]int, taskCount)
				for i := 0; i < taskCount; i++ {
					i := i
					expected[i] = i
					g.Go(func() int {
						cur := currentConcurrent.Add(1)
						if cur > int64(maxGoroutines) {
							errCount.Add(1)
						}
						time.Sleep(time.Millisecond)
						currentConcurrent.Add(-1)
						return i
					})
				}
				res := g.Wait()
				sort.Ints(res)
				require.Equal(t, expected, res)
				require.Equal(t, int64(0), errCount.Load())
				require.Equal(t, int64(0), currentConcurrent.Load())
			})
		}
	})
}
