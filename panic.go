package conc

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"
)

type PanicCatcher struct {
	caught atomic.Value
}

// Try executes f, catching any panic it might spawn. It is safe
// to call from multiple goroutines simultaneously.
func (p *PanicCatcher) Try(f func()) {
	defer func() {
		if val := recover(); val != nil {
			var callers [32]uintptr
			n := runtime.Callers(1, callers[:])
			p.caught.CompareAndSwap(nil, &CaughtPanic{
				Value:   val,
				Callers: callers[:n],
				Stack:   debug.Stack(),
			})
		}
	}()
	f()
}

// Propagate panics if any calls to Try caught a panic. It will
// panic with the value of the first panic caught.
func (p *PanicCatcher) Propagate() {
	if p.caught.Load() != nil {
		panic(p.caught)
	}
}

// Value returns the value of the first panic caught by Try, or nil if
// no calls to Try panicked
func (p *PanicCatcher) Value() *CaughtPanic {
	val := p.caught.Load()
	if val == nil {
		return nil
	}
	return val.(*CaughtPanic)
}

type CaughtPanic struct {
	Value   any
	Callers []uintptr
	Stack   []byte
}

func (c *CaughtPanic) Error() string {
	return fmt.Sprintf("original value: %q\nstacktrace: %s", c.Value, c.Stack)
}
