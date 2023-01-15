package examples

import (
	"runtime/debug"
	"testing"

	"github.com/sourcegraph/conc"
	"github.com/stretchr/testify/require"
)

func doSomethingThatMightPanic() {
	panic("big bad")
}

type recoveredPanic struct {
	val   any
	stack []byte
}

func example1_stdlib() {
	done := make(chan *recoveredPanic)
	go func() {
		defer func() {
			if v := recover(); v != nil {
				done <- &recoveredPanic{
					val:   v,
					stack: debug.Stack(),
				}
			} else {
				done <- nil
			}
		}()
		doSomethingThatMightPanic()
	}()
	err := <-done
	if err != nil {
		panic(err)
	}
}

func example1_conc() {
	var wg conc.WaitGroup
	wg.Go(doSomethingThatMightPanic)
	// panics with a nice stacktrace
	wg.Wait()
}

func TestExample1(t *testing.T) {
	require.Panics(t, example1_stdlib)
	require.Panics(t, example1_conc)
}
