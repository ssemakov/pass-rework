package service

import "sync/atomic"

// ShouldQuit never waits on the UI thread. Clipboard dispatch may need that
// thread while Close drains the database worker and clears owned clipboard data.
type Shutdown struct {
	started atomic.Bool
	ready   atomic.Bool
	close   func()
	quit    func()
}

func NewShutdown(close, quit func()) *Shutdown { return &Shutdown{close: close, quit: quit} }
func (s *Shutdown) ShouldQuit() bool {
	if s.ready.Load() {
		return true
	}
	if s.started.CompareAndSwap(false, true) {
		go func() { s.close(); s.ready.Store(true); s.quit() }()
	}
	return false
}
