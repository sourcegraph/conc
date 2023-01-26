package panics

// Try executes f, catching and returning any panic it might spawn.
//
// The recovered panic can be propagated with panic(), or handled as a normal error with
// (*RecoveredPanic).AsError().
func Try(f func()) *RecoveredPanic {
	var c Catcher
	c.Try(f)
	return c.Recovered()
}
