package stream

import (
	"github.com/camdencheek/conc"
	"github.com/camdencheek/conc/pool"
)

func New(maxGoroutines int) Stream {
	s := Stream{
		pool:  pool.New().WithMaxGoroutines(maxGoroutines),
		free:  make(chan callbackCh, maxGoroutines+1),
		queue: make(chan callbackCh, maxGoroutines+1),
	}
	// Pre-populate the free list with channels
	for i := 0; i < cap(s.free); i++ {
		s.free <- make(callbackCh, 1)
	}

	// Start the callbacker
	s.callbackerHandle.Go(s.callbacker)

	return s
}

type Stream struct {
	pool             pool.Pool
	callbackerHandle conc.WaitGroup
	free             chan callbackCh
	queue            chan callbackCh
}

func (s *Stream) Do(f func(), callback func()) {
	ch := <-s.free
	s.queue <- ch
	s.pool.Do(func() {
		defer func() { ch <- callback }()
		f()
	})
}

func (s *Stream) Wait() {
	s.pool.Wait()
	close(s.queue)
	s.callbackerHandle.Wait()
}

func (s *Stream) callbacker() {
	for callbackCh := range s.queue {
		callback := <-callbackCh
		callback()
		s.free <- callbackCh
	}
}

type callbackCh chan func()

func Submit[T, R any](s *Stream, f func() T, callback func(T)) {
	var res T
	newF := func() {
		res = f()
	}
	newCallback := func() {
		callback(res)
	}
	s.Do(newF, newCallback)
}

func SubmitErr[T, R any](s *Stream, f func() (T, error), callback func(T, error)) {
	var (
		res T
		err error
	)
	newF := func() {
		res, err = f()
	}
	newCallback := func() {
		callback(res, err)
	}
	s.Do(newF, newCallback)
}
