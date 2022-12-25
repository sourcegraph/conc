package conc

// Limiter is a simple channel-based semaphore.
func NewLimiter(n int) Limiter {
	return make(Limiter, n)
}

type Limiter chan struct{}

func (l Limiter) Limit() int {
	return cap(l)
}

func (l Limiter) Acquire() {
	if l != nil {
		l <- struct{}{}
	}
}

func (l Limiter) Release() {
	if l != nil {
		<-l
	}
}
