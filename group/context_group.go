package group

import (
	"context"
)

type ContextGroup struct {
	errGroup ErrorGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func (g *ContextGroup) Go(f func(ctx context.Context) error) {
	g.errGroup.Go(func() error {
		err := f(g.ctx)
		if err != nil && g.cancel != nil {
			// Add the error directly because otherwise, canceling could cause
			// another goroutine to exit and return an error before this error
			// was added, which breaks the expectations of WithFirstError().
			g.errGroup.addErr(err)
			g.cancel()
			return nil
		}
		return err
	})
}

func (g *ContextGroup) Wait() error {
	return g.errGroup.Wait()
}

func (g *ContextGroup) WithCancelOnError() *ContextGroup {
	g.ctx, g.cancel = context.WithCancel(g.ctx)
	return g
}

func (g *ContextGroup) WithMaxConcurrency(limit int) *ContextGroup {
	g.errGroup = g.errGroup.WithMaxConcurrency(limit)
	return g
}

func (g *ContextGroup) WithFirstError() *ContextGroup {
	g.errGroup = g.errGroup.WithFirstError()
	return g
}
