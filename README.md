# `conc`: better structured concurrency for go

`conc` is your toolbelt for structured concurrency in go, making common tasks
easier and safer.

The main goals of the package are:
1) Make it harder to leak goroutines
2) Handle panics gracefully
3) Make concurrent code easier to read

## Goal #1: Make it harder to leak goroutines

A common pain point when working with goroutines is cleaning them up. It's
really easy to fire off a `go` statement and fail to properly wait for it to
complete.

`conc` takes the opinionated stance that all concurrency should be scoped.
That is, goroutines should have an owner and that owner should always
ensure that its owned goroutines exit properly.

In `conc`, the owner of a goroutine is always a `conc.WaitGroup`. Goroutines
are spawned in a `WaitGroup` with `(*WaitGroup).Go()`, and
`(*WaitGroup).Wait()` should always be called before the `WaitGroup` goes out
of scope.

In some cases, you might want a spawned goroutine to outlast the scope of the
caller. In that case, you could pass a `WaitGroup` into the spawning function.

```go
func main() {
    var wg conc.WaitGroup
    defer wg.Wait()

    startTheThing(&wg)
}

func startTheThing(wg *conc.WaitGroup) {
    wg.Go(func() { ... })
}
```

For some more discussion on why scoped concurrency is nice, check out [this
blog
post](https://vorpus.org/blog/notes-on-structured-concurrency-or-go-statement-considered-harmful/).

## Goal #2: Handle panics gracefully

A frequent problem with goroutines in long-running applications is handling
panics. A goroutine spawned without a panic handler will crash the whole process
on panic. This is usually undesirable. 

However, if you do add a panic handler to a goroutine, what do you do with the
panic once you catch it? Some options:
1) Ignore it
2) Log it
3) Turn it into an error and return that to the goroutine spawner
4) Propagate the panic to the goroutine spawner

Ignoring panics is a bad idea since panics usually mean there is actually
something wrong and someone should fix it.

Just logging panics isn't great either because then there is no indication to the spawner
that something bad happened, and it might just continue on as normal even though your
program is in a really bad state.

Both (3) and (4) are reasonable options, but both require the goroutine to have
an owner that can actually receive the message that something went wrong. This
is generally not true with a goroutine spawned with `go`, but in the `conc`
package, all goroutines have an owner that must collect the spawned goroutine. 
In the conc package, any call to `Wait()` will panic if any of the spawned goroutines
panicked. Additionally, it decorates the panic value with a stacktrace from the child
goroutine so that you don't lose information about what caused the panic.

Doing this all correctly every time you spawn something with `go` is not
trivial and it requires a lot of boilerplate that makes the important parts of
the code more difficult to read, so `conc` does this for you.

<table>
    <tr>
        <th>
        `stdlib`
        </th>
        <th>
        `conc`
        </th>
    </tr>
    <tr>
        <td>
```go
type caughtPanicError struct {
	val   any
	stack []byte
}

func (e *caughtPanicError) Error() string {
	return fmt.Sprintf("panic: %q\n%s", e.val, string(e.stack))
}

func spawn() {
	done := make(chan error)
	go func() {
		defer func() {
			if val := recover(); val != nil {
				done <- caughtPanicError{
					val: val, 
					stack: debug.Stack()
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
```
        </td>
        <td>
```go
func spawn() {
    var wg conc.WaitGroup
    wg.Go(doSomethingThatMightPanic)
    wg.Wait()
}
```
        </td>
    </tr>
</table>

## Goal #3: Make concurrent code easier to read

Doing concurrency correctly is difficult. Doing it in a way that doesn't
obfuscate what the code is actually doing is more difficult. The `conc` package
attempts to make common operations easier by abstracting as much boilerplate
complexity as possible.

Want to run a set of concurrent tasks with a bounded set of goroutines? Use
`pool.New()`. Want to process an ordered stream of results concurrently, but
still maintain order? Try `stream.New()`. What about a concurrent map over
a slice? Take a peek at `iter.Map()`.

Browse some examples below for some comparisons with doing these by hand.

# Examples

Each of these examples forgoes propagating panics for simplicity. To see
what kind of complexity that would add, check out the "Goal #2" header above.

Spawn a set of goroutines and waiting for them to finish:

<table>
    <tr>
        <th>
        `stdlib`
        </th>
        <th>
        `conc`
        </th>
    </tr>
    <tr>
        <td>
```go
func main() {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// if doSomething panics, the process crashes!
			doSomething()
		}()
	}
	wg.Wait()
}
```
        </td>
        <td>
```go
func main() {
	var wg conc.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Go(doSomething)
	}
	wg.Wait()
}
```
        </td>
    </tr>
</table>

Process each element of a stream in a static pool of goroutines:

<table>
    <tr>
        <th>
        `stdlib`
        </th>
        <th>
        `conc`
        </th>
    </tr>
    <tr>
        <td>
```go
func process(stream chan int) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for elem := range stream {
				handle(elem)
			}
		}()
	}
	wg.Wait()
}
```
        </td>
        <td>
```go
func process(stream chan int) {
	p := pool.New().WithMaxGoroutines(10)
	for elem := range stream {
		p.Go(func() {
			handle(values[i])
		})
	}
	p.Wait()
}
```
        </td>
    </tr>
</table>

Process each element of a slice in a static pool of goroutines:

<table>
    <tr>
        <th>
        `stdlib`
        </th>
        <th>
        `conc`
        </th>
    </tr>
    <tr>
        <td>
```go
func main() {
	values := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	feeder := make(chan int, 8)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for elem := range feeder {
				handle(elem)
			}
		}()
	}

	for _, value := range values {
		feeder <- value
	}

	wg.Wait()
}
```
        </td>
        <td>
```go
func main() {
	values := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	iter.ForEach(values, handle)
}
```
        </td>
    </tr>
</table>

Process an ordered stream concurrently:


<table>
    <tr>
        <th>
        `stdlib`
        </th>
        <th>
        `conc`
        </th>
    </tr>
    <tr>
        <td>
```go
func mapStream(input chan int, output chan int, f func(int) int) {
	tasks := make(chan func())
	taskResults := make(chan chan int)

	// Spawn the worker goroutines
	var workerWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for task := range tasks {
				task()
			}
		}()
	}

	// Spawn the goroutine that reads results in order
	var readerWg sync.WaitGroup
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for taskResult := range taskResults {
			output <- taskResult
		}
	}

	// Feed the workers with tasks
	for elem := range input {
		resultCh := make(chan int, 1)
		taskResults <- resultCh
		tasks <- func() {
			resultCh <- f(elem)
		}
	}

	// We've exhausted input. Wait for everything to finish
	close(tasks)
	workerWg.Wait()
	close(taskResults)
	readerWg.Wait()
}
```
        </td>
        <td>
```go
func mapStream(input chan int, output chan int, f func(int) int) {
	s := stream.New().WithMaxGoroutines(10)
	for elem := range input {
		elem := elem
		s.Go(func() {
			res := f(elem)
			return func() { output <- res }
		})
	}
	s.Wait()
}
```
        </td>
    </tr>
</table>
