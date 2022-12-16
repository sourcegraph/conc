package pool

import (
	"context"
)

type ContextPool struct {
	ErrorPool

	ctx    context.Context
	cancel context.CancelFunc
}

func (g *ContextPool) Do(f func(ctx context.Context) error) {
	g.ErrorPool.Go(func() error {
		err := f(g.ctx)
		if err != nil && g.cancel != nil {
			// Add the error directly because otherwise, canceling could cause
			// another goroutine to exit and return an error before this error
			// was added, which breaks the expectations of WithFirstError().
			g.ErrorPool.addErr(err)
			g.cancel()
			return nil
		}
		return err
	})
}

func (p *ContextPool) Wait() error {
	return p.ErrorPool.Wait()
}

func (p *ContextPool) WithCancelOnError() *ContextPool {
	p.ctx, p.cancel = context.WithCancel(p.ctx)
	return p
}

func (p *ContextPool) WithMaxGoroutines(limit int) *ContextPool {
	p.ErrorPool = *p.ErrorPool.WithMaxGoroutines(limit)
	return p
}
