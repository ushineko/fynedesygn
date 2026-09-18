package forms

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

// DefaultSaveDelay is how long a Saver waits after the last change.
const DefaultSaveDelay = time.Second

/*
Saver coalesces writes: every change schedules one save a moment after the
last change, so a slider dragged across a range or a form typed into is one
write to the file rather than one per keystroke. Flush writes a pending save
at once, for quitting.

The save runs on the UI thread through fyne.Do, because it is the program's
document being written and the program reads that document from its
builders. Under the Fyne test driver fyne.Do runs inline, so a headless test
sees the save when the timer fires.
*/
type Saver struct {
	delay time.Duration
	save  func()

	mu      sync.Mutex
	seq     int
	pending bool
}

// NewSaver makes a saver calling save at most once per quiet period of delay;
// zero means DefaultSaveDelay.
func NewSaver(delay time.Duration, save func()) *Saver {
	if delay <= 0 {
		delay = DefaultSaveDelay
	}
	return &Saver{delay: delay, save: save}
}

// Schedule notes a change. The save happens delay after the last Schedule.
func (s *Saver) Schedule() {
	s.mu.Lock()
	s.seq++
	seq := s.seq
	s.pending = true
	s.mu.Unlock()
	go func() {
		time.Sleep(s.delay)
		s.mu.Lock()
		if s.seq != seq || !s.pending {
			s.mu.Unlock()
			return // a later change owns the timer, or Flush already saved
		}
		s.pending = false
		s.mu.Unlock()
		fyne.Do(s.save)
	}()
}

// Flush saves now if a save is pending. Call before quitting or restarting.
func (s *Saver) Flush() {
	s.mu.Lock()
	if !s.pending {
		s.mu.Unlock()
		return
	}
	s.pending = false
	s.seq++ // any timer in flight finds a different sequence and stops
	s.mu.Unlock()
	s.save()
}

// Pending reports whether a save is waiting for the quiet period.
func (s *Saver) Pending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending
}
