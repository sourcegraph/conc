package panics

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExampleCatcher() {
	var pc Catcher
	i := 0
	pc.Try(func() { i += 1 })
	pc.Try(func() { panic("abort!") })
	pc.Try(func() { i += 1 })

	rc := pc.Recovered()

	fmt.Println(i)
	fmt.Println(rc.Value.(string))
	// Output:
	// 2
	// abort!
}

func ExampleCatcher_callers() {
	var pc Catcher
	pc.Try(func() { panic("mayday!") })

	recovered := pc.Recovered()

	// For debugging, the pre-formatted recovered.Stack is easier to use than
	// rc.Callers. This is not used in the example because its output is
	// machine-specific.

	frames := runtime.CallersFrames(recovered.Callers)
	for {
		frame, more := frames.Next()

		fmt.Println(frame.Function)

		if !more {
			break
		}
	}
	// Output:
	// github.com/sourcegraph/conc/panics.(*Catcher).tryRecover
	// runtime.gopanic
	// github.com/sourcegraph/conc/panics.ExampleCatcher_callers.func1
	// github.com/sourcegraph/conc/panics.(*Catcher).Try
	// github.com/sourcegraph/conc/panics.ExampleCatcher_callers
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
	// main.main
	// runtime.main
	// runtime.goexit
}

func TestCatcher(t *testing.T) {
	t.Parallel()

	err1 := errors.New("SOS")

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		var pc Catcher
		pc.Try(func() { panic(err1) })
		recovered := pc.Recovered()
		require.ErrorIs(t, recovered, err1)
		require.ErrorAs(t, recovered, &err1)
		// The exact contents aren't tested because the stacktrace contains local file paths
		// and even the structure of the stacktrace is bound to be unstable over time. Just
		// test a couple of basics.
		require.Contains(t, recovered.Error(), "SOS", "error should contain the panic message")
		require.Contains(t, recovered.Error(), "panics.(*Catcher).Try", recovered.Error(), "error should contain the stack trace")
	})

	t.Run("not error", func(t *testing.T) {
		var pc Catcher
		pc.Try(func() { panic("definitely not an error") })
		recovered := pc.Recovered()
		require.NotErrorIs(t, recovered, err1)
		require.Nil(t, recovered.Unwrap())
	})

	t.Run("repanic panics", func(t *testing.T) {
		var pc Catcher
		pc.Try(func() { panic(err1) })
		require.Panics(t, pc.Repanic)
	})

	t.Run("repanic does not panic without child panic", func(t *testing.T) {
		t.Parallel()
		var pc Catcher
		pc.Try(func() { _ = 1 })
		require.NotPanics(t, pc.Repanic)
	})

	t.Run("is goroutine safe", func(t *testing.T) {
		t.Parallel()
		var wg sync.WaitGroup
		var pc Catcher
		for i := 0; i < 100; i++ {
			i := i
			wg.Add(1)
			func() {
				defer wg.Done()
				pc.Try(func() {
					if i == 50 {
						panic("50")
					}

				})
			}()
		}
		wg.Wait()
		require.Equal(t, "50", pc.Recovered().Value)
	})
}
