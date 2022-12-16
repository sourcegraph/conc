package pool

import (
	"context"
)

type ContextPool struct {
	errorPool ErrorPool

	ctx    context.Context
	cancel context.CancelFunc
}

func (g *ContextPool) Go(f func(ctx context.Context) error) {
	g.errorPool.Go(func() error {
		err := f(g.ctx)
		if err != nil && g.cancel != nil {
			// Add the error directly because otherwise, canceling could cause
			// another goroutine to exit and return an error before this error
			// was added, which breaks the expectations of WithFirstError().
			g.errorPool.addErr(err)
			g.cancel()
			return nil
		}
		return err
	})
}

func (p *ContextPool) Wait() error {
	return p.errorPool.Wait()
}

func (p *ContextPool) WithCancelOnError() *ContextPool {
	p.ctx, p.cancel = context.WithCancel(p.ctx)
	return p
}
