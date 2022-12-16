package group

import (
	"context"
	"sync"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type ErrorGroup struct {
	group Group

	onlyFirst bool

	mu   sync.Mutex
	errs error
}

func (g *ErrorGroup) Go(f func() error) {
	g.group.Go(func() {
		g.addErr(f())
	})
}

func (g *ErrorGroup) Wait() error {
	g.group.Wait()
	return g.errs
}

func (g ErrorGroup) WithMaxConcurrency(limit int) ErrorGroup {
	g.group = g.group.WithMaxConcurrency(limit)
	return g
}

func (g ErrorGroup) WithContext(ctx context.Context) ContextGroup {
	return ContextGroup{
		errGroup: g,
		ctx:      ctx,
	}
}

func (g ErrorGroup) WithFirstError() ErrorGroup {
	g.onlyFirst = true
	return g
}

func (g *ErrorGroup) addErr(err error) {
	if err != nil {
		g.mu.Lock()
		if g.onlyFirst {
			if g.errs == nil {
				g.errs = err
			}
		} else {
			g.errs = errors.Append(g.errs, err)
		}
		g.mu.Unlock()
	}
}
