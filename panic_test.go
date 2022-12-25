package conc

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExamplePanicCatcher() {
	var pc PanicCatcher
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

func ExamplePanicCatcher_callers() {
	var pc PanicCatcher
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
	// runtime.gopanic
	// github.com/camdencheek/conc.ExamplePanicCatcher_callers.func1
	// github.com/camdencheek/conc.(*PanicCatcher).Try
	// github.com/camdencheek/conc.ExamplePanicCatcher_callers
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
	// main.main
	// runtime.main
	// runtime.goexit
}

func TestPanicCatcher(t *testing.T) {
	err1 := errors.New("SOS")

	t.Run("error", func(t *testing.T) {
		var pc PanicCatcher
		pc.Try(func() { panic(err1) })
		recovered := pc.Recovered()
		require.ErrorIs(t, recovered, err1)
		require.ErrorAs(t, recovered, &err1)
	})

	t.Run("repanic panics", func(t *testing.T) {
		var pc PanicCatcher
		pc.Try(func() { panic(err1) })
		require.Panics(t, pc.Repanic)
	})

	t.Run("repanic does not panic without child panic", func(t *testing.T) {
		var pc PanicCatcher
		pc.Try(func() { _ = 1 })
		require.NotPanics(t, pc.Repanic)
	})

	t.Run("is goroutine safe", func(t *testing.T) {
		var wg sync.WaitGroup
		var pc PanicCatcher
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
