package conc

import "sync/atomic"

type PanicCatcher struct {
	panicVal atomic.Value
}

// Try executes f, catching any panic it might spawn. It is safe
// to call from multiple goroutines simultaneously.
func (p *PanicCatcher) Try(f func()) {
	defer func() {
		if r := recover(); r != nil {
			p.panicVal.CompareAndSwap(nil, r)
		}
	}()
	f()
}

// Propagate panics if any calls to Try caught a panic. It will
// panic with the value of the first panic caught.
func (p *PanicCatcher) Propagate() {
	if p.panicVal.Load() != nil {
		panic(p.panicVal)
	}
}
