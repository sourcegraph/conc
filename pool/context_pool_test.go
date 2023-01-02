package pool

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func ExampleContextPool() {
	p := New().WithContext(context.Background())
	for i := 0; i < 3; i++ {
		i := i
		p.Go(func(ctx context.Context) error {
			if i == 2 {
				return errors.New("I will cancel all other tasks!")
			}
			<-ctx.Done()
			return nil
		})
	}
	err := p.Wait()
	fmt.Println(err)
	// Output:
	// I will cancel all other tasks!
}

func TestContextPool(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")
	bgctx := context.Background()

	t.Run("behaves the same as ErrorGroup", func(t *testing.T) {
		t.Run("wait returns no error if no errors", func(t *testing.T) {
			p := New().WithContext(bgctx)
			p.Go(func(context.Context) error { return nil })
			require.NoError(t, p.Wait())
		})

		t.Run("wait errors if func returns error", func(t *testing.T) {
			p := New().WithContext(bgctx)
			p.Go(func(context.Context) error { return err1 })
			require.ErrorIs(t, p.Wait(), err1)
		})

		t.Run("wait error is all returned errors", func(t *testing.T) {
			p := New().WithErrors().WithContext(bgctx)
			p.Go(func(context.Context) error { return err1 })
			p.Go(func(context.Context) error { return nil })
			p.Go(func(context.Context) error { return err2 })
			err := p.Wait()
			require.ErrorIs(t, err, err1)
			require.ErrorIs(t, err, err2)
		})
	})

	t.Run("context error propagates", func(t *testing.T) {
		t.Run("canceled", func(t *testing.T) {
			ctx, cancel := context.WithCancel(bgctx)
			p := New().WithContext(ctx)
			p.Go(func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			})
			cancel()
			require.ErrorIs(t, p.Wait(), context.Canceled)
		})

		t.Run("timed out", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(bgctx, time.Millisecond)
			defer cancel()
			p := New().WithContext(ctx)
			p.Go(func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			})
			require.ErrorIs(t, p.Wait(), context.DeadlineExceeded)
		})
	})

	t.Run("cancel on error", func(t *testing.T) {
		p := New().WithMaxGoroutines(2).WithContext(bgctx)
		p.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		p.Go(func(ctx context.Context) error {
			return err1
		})
		err := p.Wait()
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, err, err1)
	})

	t.Run("WithFirstError", func(t *testing.T) {
		p := New().WithContext(bgctx).WithFirstError()
		p.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return err2
		})
		p.Go(func(ctx context.Context) error {
			return err1
		})
		err := p.Wait()
		require.ErrorIs(t, err, err1)
		require.NotErrorIs(t, err, context.Canceled)
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()
		for _, maxConcurrent := range []int{1, 10, 100} {
			t.Run(strconv.Itoa(maxConcurrent), func(t *testing.T) {
				t.Parallel()
				p := New().WithContext(bgctx).WithMaxGoroutines(maxConcurrent)

				var currentConcurrent atomic.Int64
				for i := 0; i < 100; i++ {
					p.Go(func(context.Context) error {
						cur := currentConcurrent.Add(1)
						if cur > int64(maxConcurrent) {
							return errors.Newf("expected no more than %d concurrent goroutine", maxConcurrent)
						}
						time.Sleep(time.Millisecond)
						currentConcurrent.Add(-1)
						return nil
					})
				}
				require.NoError(t, p.Wait())
				require.Equal(t, int64(0), currentConcurrent.Load())
			})
		}
	})
}
