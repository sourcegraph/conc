package group

import (
	"context"

	"github.com/camdencheek/conc"
)

func New() Group {
	return Group{}
}

type Group struct {
	handle conc.WaitGroup

	// A nil channel means no limit.
	// A full channel means the limit is exhausted.
	limiter chan struct{}
}

func (g *Group) Go(f func()) {
	g.acquire()
	g.handle.Spawn(func() {
		defer g.release()
		f()
	})
}

func (g *Group) acquire() {
	g.limiter <- struct{}{}
}

func (g *Group) release() {
	<-g.limiter
}

func (g *Group) Wait() {
	g.handle.Wait()
}

func (g Group) WithMaxConcurrency(limit int) Group {
	g.limiter = make(chan struct{}, limit)
	return g
}

func (g Group) WithErrors() ErrorGroup {
	return ErrorGroup{
		group: g,
	}
}

func (g Group) WithContext(ctx context.Context) ContextGroup {
	return ContextGroup{
		errGroup: g.WithErrors(),
		ctx:      ctx,
	}
}
