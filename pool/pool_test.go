package pool

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/camdencheek/conc"
	"github.com/stretchr/testify/require"
)

func TestPool(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		g := New()
		var completed atomic.Int64
		for i := 0; i < 100; i++ {
			g.Go(func() {
				time.Sleep(1 * time.Millisecond)
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
				g := New().WithMaxGoroutines(maxConcurrent)

				var currentConcurrent atomic.Int64
				var errCount atomic.Int64
				taskCount := maxConcurrent * 10
				for i := 0; i < taskCount; i++ {
					g.Go(func() {
						cur := currentConcurrent.Add(1)
						if cur > int64(maxConcurrent) {
							errCount.Add(1)
						}
						time.Sleep(time.Millisecond)
						currentConcurrent.Add(-1)
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
		require.Panics(t, g.Wait)
	})

	t.Run("panics do not exhaust goroutines", func(t *testing.T) {
		g := New().WithMaxGoroutines(2)
		for i := 0; i < 10; i++ {
			g.Go(func() {
				panic(42)
			})
		}
		require.Panics(t, g.Wait)
	})
}

func BenchmarkPool(b *testing.B) {
	b.Run("simplest", func(b *testing.B) {
		p := New()
		for i := 0; i < b.N; i++ {
			p.Go(func() {
				i := 0
				i = 1
				_ = i
			})
		}
		p.Wait()
	})

	b.Run("atomic increment", func(b *testing.B) {
		p := New()
		var j atomic.Int64
		for i := 0; i < b.N; i++ {
			p.Go(func() {
				j.Add(1)
			})
		}
		p.Wait()
		_ = j.Load()
	})

	b.Run("atomic increment preallocated closure", func(b *testing.B) {
		p := New()
		var j atomic.Int64
		f := func() { j.Add(1) }
		for i := 0; i < b.N; i++ {
			p.Go(f)
		}
		p.Wait()
		_ = j.Load()
	})

	b.Run("conc", func(b *testing.B) {
		var p conc.WaitGroup
		var j atomic.Int64
		for i := 0; i < b.N; i++ {
			p.Go(func() { j.Add(1) })
		}
		p.Wait()
		_ = j.Load()
	})

}
