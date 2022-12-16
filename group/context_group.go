package group

import (
	"context"
)

type ContextGroup struct {
	ErrorGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func (g *ContextGroup) Go(f func(ctx context.Context) error) {
	g.ErrorGroup.Go(func() error {
		err := f(g.ctx)
		if err != nil && g.cancel != nil {
			// Add the error directly because otherwise, canceling could cause
			// another goroutine to exit and return an error before this error
			// was added, which breaks the expectations of WithFirstError().
			g.ErrorGroup.addErr(err)
			g.cancel()
			return nil
		}
		return err
	})
}

func (g *ContextGroup) Wait() error {
	return g.ErrorGroup.Wait()
}

func (g *ContextGroup) WithCancelOnError() *ContextGroup {
	g.ctx, g.cancel = context.WithCancel(g.ctx)
	return g
}
