package pool

type limiter chan struct{}

func (l limiter) Limit() int {
	return cap(l)
}

func (l limiter) Acquire() {
	l <- struct{}{}
}

func (l limiter) Release() {
	<-l
}
