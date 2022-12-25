package group

import (
	"runtime"
	"strconv"
	"sync"
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
		require.Panics(t, func() { g.Wait() })
	})
}

func BenchmarkGroup(b *testing.B) {
	g := New()
	for i := 0; i < b.N; i++ {
		g.Go(func() {
			i := 0
			i = 1
			_ = i
		})
	}
	g.Wait()
}

func BenchmarkGroup2(b *testing.B) {
	g := New()
	var ai atomic.Uint32
	for i := 0; i < b.N; i++ {
		g.Go(func() {
			ai.Add(1)
		})
	}
	g.Wait()
}

func BenchmarkGroup21(b *testing.B) {
	g := New().WithMaxGoroutines(runtime.GOMAXPROCS(0))
	for i := 0; i < b.N; i++ {
		g.Go(func() {
			time.Sleep(10 * time.Nanosecond)
		})
	}
	g.Wait()
}

func BenchmarkGroup22(b *testing.B) {
	for i := 0; i < b.N; i++ {
		g := New().WithMaxGoroutines(runtime.GOMAXPROCS(0))
		var ai atomic.Int32
		f := func() {
			time.Sleep(100 * time.Nanosecond)
			ai.Add(1)
			time.Sleep(100 * time.Nanosecond)
		}
		for j := 0; j < 500; j++ {
			g.Go(f)
		}
		g.Wait()
	}
}

func BenchmarkGroup3(b *testing.B) {
	g := New().WithMaxGoroutines(runtime.GOMAXPROCS(0))
	var ai atomic.Uint32
	for i := 0; i < b.N; i++ {
		ai.Add(1)
	}
	g.Wait()
}

func BenchmarkGroup4(b *testing.B) {
	var wg sync.WaitGroup
	var ai atomic.Uint32
	f := func() {
		defer wg.Done()
		ai.Add(1)
	}
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go f()
	}
	wg.Wait()
}
