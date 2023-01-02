package conc

// Limiter is a simple, goroutine-safe, channel-based semaphore.
func NewLimiter(n int) Limiter {
	return make(Limiter, n)
}

type Limiter chan struct{}

func (l Limiter) Limit() int {
	return cap(l)
}

func (l Limiter) Acquire() {
	l <- struct{}{}
}

func (l Limiter) Release() {
	<-l
}
