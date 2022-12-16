package group

import (
	"context"
)

type ContextGroup struct {
	errorGroup ErrorGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func (g *ContextGroup) Go(f func(ctx context.Context) error) {
	g.errorGroup.Go(func() error {
		err := f(g.ctx)
		if err != nil && g.cancel != nil {
			// Add the error directly because otherwise, canceling could cause
			// another goroutine to exit and return an error before this error
			// was added, which breaks the expectations of WithFirstError().
			g.errorGroup.addErr(err)
			g.cancel()
			return nil
		}
		return err
	})
}

func (g *ContextGroup) Wait() error {
	return g.errorGroup.Wait()
}

func (g *ContextGroup) WithFirstError() *ContextGroup {
	g.errorGroup.WithFirstError()
	return g
}

func (g *ContextGroup) WithMaxGoroutines(n int) *ContextGroup {
	g.errorGroup.WithMaxGoroutines(n)
	return g
}

func (g *ContextGroup) WithCancelOnError() *ContextGroup {
	g.ctx, g.cancel = context.WithCancel(g.ctx)
	return g
}
