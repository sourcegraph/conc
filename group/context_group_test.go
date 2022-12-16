package group

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func TestContextErrorGroup(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")

	t.Run("behaves the same as ErrorGroup", func(t *testing.T) {
		bgctx := context.Background()
		t.Run("wait returns no error if no errors", func(t *testing.T) {
			g := New().WithContext(bgctx)
			g.Go(func(context.Context) error { return nil })
			require.NoError(t, g.Wait())
		})

		t.Run("wait errors if func returns error", func(t *testing.T) {
			g := New().WithContext(bgctx)
			g.Go(func(context.Context) error { return err1 })
			require.ErrorIs(t, g.Wait(), err1)
		})

		t.Run("wait error is all returned errors", func(t *testing.T) {
			g := New().WithErrors().WithContext(bgctx)
			g.Go(func(context.Context) error { return err1 })
			g.Go(func(context.Context) error { return nil })
			g.Go(func(context.Context) error { return err2 })
			err := g.Wait()
			require.ErrorIs(t, err, err1)
			require.ErrorIs(t, err, err2)
		})
	})

	t.Run("context error propagates", func(t *testing.T) {
		t.Run("canceled", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			g := New().WithContext(ctx)
			g.Go(func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			})
			cancel()
			require.ErrorIs(t, g.Wait(), context.Canceled)
		})

		t.Run("timed out", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
			defer cancel()
			g := New().WithContext(ctx)
			g.Go(func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			})
			require.ErrorIs(t, g.Wait(), context.DeadlineExceeded)
		})
	})

	t.Run("CancelOnError", func(t *testing.T) {
		g := New().WithContext(context.Background()).WithCancelOnError()
		g.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		g.Go(func(ctx context.Context) error {
			return err1
		})
		require.ErrorIs(t, g.Wait(), context.Canceled)
		require.ErrorIs(t, g.Wait(), err1)
	})

	t.Run("WithFirstError", func(t *testing.T) {
		g := New().WithContext(context.Background()).WithCancelOnError().WithFirstError()
		g.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return err2
		})
		g.Go(func(ctx context.Context) error {
			return err1
		})
		require.ErrorIs(t, g.Wait(), err1)
		require.NotErrorIs(t, g.Wait(), context.Canceled)
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()
		for _, maxConcurrent := range []int{1, 10, 100} {
			t.Run(strconv.Itoa(maxConcurrent), func(t *testing.T) {
				t.Parallel()
				ctx := context.Background()
				g := New().WithContext(ctx).WithMaxGoroutines(maxConcurrent)

				var currentConcurrent atomic.Int64
				for i := 0; i < 100; i++ {
					g.Go(func(context.Context) error {
						cur := currentConcurrent.Add(1)
						if cur > int64(maxConcurrent) {
							return errors.Newf("expected no more than %d concurrent goroutine", maxConcurrent)
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
