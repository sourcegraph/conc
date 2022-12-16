package group

import (
	"context"

	"github.com/camdencheek/conc"
)

func New() *Group {
	return &Group{}
}

type Group struct {
	handle  conc.WaitGroup
	limiter conc.Limiter
}

func (g *Group) Go(f func()) {
	g.limiter.Acquire()
	g.handle.Go(func() {
		defer g.limiter.Release()
		f()
	})
}

func (g *Group) Wait() {
	g.handle.Wait()
}

func (g *Group) MaxGoroutines() int {
	return g.limiter.Limit()
}

func (g *Group) WithMaxGoroutines(limit int) *Group {
	g.limiter = conc.NewLimiter(limit)
	return g
}

func (g *Group) WithErrors() *ErrorGroup {
	return &ErrorGroup{
		group: *g,
	}
}

func (g *Group) WithContext(ctx context.Context) *ContextGroup {
	return &ContextGroup{
		errorGroup: *g.WithErrors(),
		ctx:        ctx,
	}
}
