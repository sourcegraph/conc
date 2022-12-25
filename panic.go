package conc

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"
)

// PanicCatcher is used to catch panics. You can execute a function with Try,
// which will catch any spawned panic. Try can be called any number of times,
// from any number of goroutines. Once all calls to Try have completed, you can
// get the value of the first panic (if any) with Value(), or you can just
// propagate the panic (re-panic) with Propagate()
type PanicCatcher struct {
	recovered atomic.Pointer[RecoveredPanic]
}

// Try executes f, catching any panic it might spawn. It is safe
// to call from multiple goroutines simultaneously.
func (p *PanicCatcher) Try(f func()) {
	defer func() {
		if val := recover(); val != nil {
			var callers [64]uintptr
			n := runtime.Callers(1, callers[:])
			p.recovered.CompareAndSwap(nil, &RecoveredPanic{
				Value:   val,
				Callers: callers[:n],
				Stack:   debug.Stack(),
			})
		}
	}()
	f()
}

// Propagate panics if any calls to Try caught a panic. It will
// panic with the value of the first panic caught, wrapped with
// caller information.
func (p *PanicCatcher) Propagate() {
	if val := p.Recovered(); val != nil {
		panic(val)
	}
}

// Recovered returns the value of the first panic caught by Try, or nil if
// no calls to Try panicked.
func (p *PanicCatcher) Recovered() *RecoveredPanic {
	return p.recovered.Load()
}

// RecoveredPanic is a panic that was caught with recover().
type RecoveredPanic struct {
	// The original value of the panic
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
	return fmt.Sprintf("panic: %q\nstacktrace:\n%s\n", c.Value, c.Stack)
}

func (c *RecoveredPanic) Unwrap() error {
	if err, ok := c.Value.(error); ok {
		return err
	}
	return nil
}
