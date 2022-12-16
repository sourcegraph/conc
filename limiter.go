package conc

func NewLimiter(n int) Limiter {
	return make(Limiter, n)
}

type Limiter chan struct{}

func (l Limiter) Acquire() {
	l <- struct{}{}
}

func (l Limiter) Release() {
	<-l
}
