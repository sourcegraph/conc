package stream

import (
	"github.com/camdencheek/conc"
	"github.com/camdencheek/conc/pool"
	"sync"
)

func New() *Stream {
	return &Stream{
		pool: *pool.New(),
	}
}

type StreamTask func() Callback
type Callback func()

type Stream struct {
	pool             pool.Pool
	callbackerHandle conc.WaitGroup
	free             chan callbackCh
	queue            chan callbackCh

	initOnce sync.Once
}

func (s *Stream) Do(f StreamTask) {
	s.initOnce.Do(s.init)

	ch := <-s.free
	s.queue <- ch
	s.pool.Go(func() {
		ch <- f()
	})
}

func (s *Stream) Wait() {
	s.initOnce.Do(s.init)

	s.pool.Wait()
	close(s.queue)
	s.callbackerHandle.Wait()
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

func (s *Stream) callbacker() {
	for callbackCh := range s.queue {
		callback := <-callbackCh
		callback()
		s.free <- callbackCh
	}
}

type callbackCh chan func()
