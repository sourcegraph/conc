// Package stream provides a concurrent, ordered stream implementation.
package stream

import (
	"sync"

	"github.com/camdencheek/conc"
	"github.com/camdencheek/conc/pool"
)

// Create a new Stream with default settings
func New() *Stream {
	return &Stream{
		pool: *pool.New(),
	}
}

// Stream is used to execute a stream of tasks concurrently while maintaining
// the order of the results.
//
// To use a stream, you submit some number of `StreamTask`s, each of which
// return a callback. Each task will be executed concurrently in the stream's
// associated Pool, and the callbacks will be executed sequentially in the
// order the tasks were submitted.
//
// Once all your tasks have been submitted, Wait() must be called to clean up
// running goroutines and propagate any panics.
//
// In the case of panic during execution of a task or a callback, all other
// tasks and callbacks will still execute. The panic will be propagated to the
// caller when Wait() is called.
type Stream struct {
	pool             pool.Pool
	callbackerHandle conc.WaitGroup
	free             freePool
	queue            chan callbackCh

	initOnce sync.Once
}

// Stream task is a task that is submitted to the stream. Submitted tasks will
// be executed concurrently. It returns a callback that will be called after
// the task has completed.
type StreamTask func() Callback

// Callback is a function that is returned by a StreamTask. Callbacks are
// called in the same order that tasks are submitted.
type Callback func()

// Go schedules a task to be run in the stream's pool. All submitted tasks
// will be executed concurrently in worker goroutines. Then, the callbacks
// returned by the tasks will be executed in the order that the tasks were
// submitted. All callbacks will be executed by the same goroutine, so no
// synchronization is necessary between callbacks.
func (s *Stream) Go(f StreamTask) {
	s.initOnce.Do(s.init)

	// Get a channel from the cache
	ch := s.free.get()

	// Queue the channel for the callbacker
	s.queue <- ch

	// Submit the task for execution
	s.pool.Go(func() {
		defer func() {
			// In the case of a panic from f, we don't want the callbacker to
			// starve waiting for a callback from this channel, so give it an
			// empty callback.
			if r := recover(); r != nil {
				ch <- func() {}
				panic(r)
			}
		}()

		// Run the task, sending its callback down this task's channel
		callback := f()
		ch <- callback
	})
}

// Wait signals to the stream that all tasks have been submitted. Wait will
// not return until all tasks and callbacks have been run.
func (s *Stream) Wait() {
	// Defer the callbacker cleanup so that it occurs even in the case
	// that one of the tasks panics and is propagated up by s.pool.Wait()
	defer func() {
		// queue may be nil if Go was never called and init never ran
		if s.queue != nil {
			close(s.queue)
		}
		s.callbackerHandle.Wait()
	}()

	// Wait for all the workers to exit
	s.pool.Wait()
}

func (s *Stream) WithMaxGoroutines(n int) *Stream {
	s.pool.WithMaxGoroutines(n)
	return s
}

func (s *Stream) init() {
	s.free = make(chan callbackCh, s.pool.MaxGoroutines()+1)
	s.queue = make(chan callbackCh, s.pool.MaxGoroutines()+1)

	// Pre-populate the free list with channels
	for i := 0; i < cap(s.free); i++ {
		s.free <- make(callbackCh, 1)
	}

	// Start the callbacker
	s.callbackerHandle.Go(s.callbacker)
}

// callbacker is responsible for calling the returned callbacks in the order
// they were submitted. There is only a single instance of callbacker running.
func (s *Stream) callbacker() {
	var panicCatcher conc.PanicCatcher
	defer panicCatcher.MaybePanic()

	// For every scheduled task, read that tasks channel from the queue.
	for callbackCh := range s.queue {
		// Wait for the task to complete and get its callback from the channel
		callback := <-callbackCh

		// Execute the callback (with panic protection)
		panicCatcher.Try(callback)

		// Return the channel to the pool of unused channels
		s.free.put(callbackCh)
	}
}

type callbackCh chan func()

type freePool chan callbackCh

func (fp freePool) get() callbackCh {
	return <-fp
}

func (fp freePool) put(ch callbackCh) {
	fp <- ch
}
