package conc

func NewLimiter(n int) Limiter {
	if n < 1 {
		return nil
	}
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
