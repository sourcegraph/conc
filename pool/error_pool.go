package pool

import (
	"context"
	"sync"

	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type ErrorPool struct {
	pool Pool

	onlyFirst bool

	mu   sync.Mutex
	errs error
}

func (p *ErrorPool) Go(f func() error) {
	p.pool.Go(func() {
		p.addErr(f())
	})
}

func (p *ErrorPool) Wait() error {
	p.pool.Wait()
	return p.errs
}

func (p *ErrorPool) WithContext(ctx context.Context) *ContextPool {
	return &ContextPool{
		errorPool: *p,
		ctx:       ctx,
	}
}

func (p *ErrorPool) WithFirstError() *ErrorPool {
	p.onlyFirst = true
	return p
}

func (p *ErrorPool) addErr(err error) {
	if err != nil {
		p.mu.Lock()
		if p.onlyFirst {
			if p.errs == nil {
				p.errs = err
			}
		} else {
			p.errs = errors.Append(p.errs, err)
		}
		p.mu.Unlock()
	}
}
