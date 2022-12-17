package conc

type PanicCatcher struct {
	panicVal any
}

func (p *PanicCatcher) Try(f func()) {
	defer func() {
		if r := recover(); r != nil && p.panicVal == nil {
			p.panicVal = r
		}
	}()
	f()
}

func (p *PanicCatcher) MaybePanic() {
	if p.panicVal != nil {
		panic(p.panicVal)
	}
}
