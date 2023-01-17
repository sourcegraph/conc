package panics

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"
)

// Catcher is used to catch panics. You can execute a function with Try,
// which will catch any spawned panic. Try can be called any number of times,
// from any number of goroutines. Once all calls to Try have completed, you can
// get the value of the first panic (if any) with Recovered(), or you can just
// propagate the panic (re-panic) with Repanic().
type Catcher struct {
	recovered atomic.Pointer[RecoveredPanic]
}

// Try executes f, catching any panic it might spawn. It is safe
// to call from multiple goroutines simultaneously.
func (p *Catcher) Try(f func()) {
	defer p.tryRecover()
	f()
}

func (p *Catcher) tryRecover() {
	if val := recover(); val != nil {
		rp := NewRecoveredPanic(1, val)
		p.recovered.CompareAndSwap(nil, &rp)
	}
}

// Repanic panics if any calls to Try caught a panic. It will panic with the
// value of the first panic caught, wrapped in a RecoveredPanic with caller
// information.
func (p *Catcher) Repanic() {
	if val := p.Recovered(); val != nil {
		panic(val)
	}
}

// Recovered returns the value of the first panic caught by Try, or nil if
// no calls to Try panicked.
func (p *Catcher) Recovered() *RecoveredPanic {
	return p.recovered.Load()
}

// NewRecoveredPanic creates a RecoveredPanic from a panic value and a
// collected stacktrace. The skip parameter allows the caller to skip stack
// frames when collecting the stacktrace. Calling with a skip of 0 means
// include the call to NewRecoveredPanic in the stacktrace.
func NewRecoveredPanic(skip int, value any) RecoveredPanic {
	// 64 frames should be plenty
	var callers [64]uintptr
	n := runtime.Callers(skip+1, callers[:])
	return RecoveredPanic{
		Value:   value,
		Callers: callers[:n],
		Stack:   debug.Stack(),
	}
}

// RecoveredPanic is a panic that was caught with recover().
type RecoveredPanic struct {
	// The original value of the panic.
	Value any
	// The caller list as returned by runtime.Callers when the panic was
	// recovered. Can be used to produce a more detailed stack information with
	// runtime.CallersFrames.
	Callers []uintptr
	// The formatted stacktrace from the goroutine where the panic was recovered.
	// Easier to use than Callers.
	Stack []byte
}

func (c *RecoveredPanic) Error() string {
	return fmt.Sprintf("panic: %v\nstacktrace:\n%s\n", c.Value, c.Stack)
}

func (c *RecoveredPanic) Unwrap() error {
	if err, ok := c.Value.(error); ok {
		return err
	}
	return nil
}
