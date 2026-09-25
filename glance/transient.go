package glance

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

/*
Transient shows a glance window for a while and hides it again: an
indicator that appears when something changes and goes away on its own.

Show restarts the timer, so changes that arrive close together keep the
window up rather than making it blink. Hide takes it down at once. Both run
on the UI thread; the timer's own hide hops there with fyne.Do.

What an indicator draws is the program's: a Meter, a Row, or a card. What it
needs from the desktop, a window that has no border, stays above the rest,
keeps out of the taskbar and never takes the focus, comes from a kwin.Rule
that matches its Title.
*/
type Transient struct {
	win  *Window
	hold time.Duration

	mu    sync.Mutex
	timer *time.Timer
	shown bool
	// winMu orders the calls on the window itself. Under the test driver
	// fyne.Do runs inline, so the timer's Hide and the next Show run on
	// different goroutines, and the window is not safe for that.
	winMu sync.Mutex
	// gen counts Shows. A timer that fires after a later Show has started
	// finds the generation moved on and does nothing, which is what stops a
	// change that arrived as the hold ran out from being hidden at once.
	gen uint64
	// OnShow runs before the window is shown, on the UI thread: the moment to
	// move the window to where the reader is looking.
	OnShow func()
}

// DefaultHold is how long an indicator stays after the last change.
const DefaultHold = 1500 * time.Millisecond

// NewTransient wraps a window that is not shown yet. A zero hold means
// DefaultHold.
func NewTransient(w *Window, hold time.Duration) *Transient {
	if hold <= 0 {
		hold = DefaultHold
	}
	return &Transient{win: w, hold: hold}
}

// Show shows the window, or keeps it up, for the hold from now.
func (t *Transient) Show() { t.ShowFor(t.hold) }

// ShowFor is Show with a hold of its own: a change worth a longer look, such
// as a switch of output beside a step of the volume, keeps the window up
// longer. A hold of zero or less is the window's own.
func (t *Transient) ShowFor(hold time.Duration) {
	if hold <= 0 {
		hold = t.hold
	}
	t.mu.Lock()
	if t.timer != nil {
		t.timer.Stop()
	}
	t.gen++
	gen := t.gen
	first := !t.shown
	t.shown = true
	t.mu.Unlock()

	if first {
		if t.OnShow != nil {
			t.OnShow()
		}
		t.winMu.Lock()
		t.win.panel.Resize()
		t.win.win.Show()
		t.winMu.Unlock()
	}

	// Armed after the window's own Show, so that the timer's hide is ordered
	// after everything Show wrote. Under the test driver fyne.Do runs inline
	// on the timer's goroutine, and a timer armed first would read the window
	// with no ordering against those writes.
	t.mu.Lock()
	if t.gen == gen {
		t.timer = time.AfterFunc(hold, func() { fyne.Do(func() { t.expire(gen) }) })
	}
	t.mu.Unlock()
}

// expire is the timer's hide: only for the run that armed it.
func (t *Transient) expire(gen uint64) {
	t.mu.Lock()
	current := t.gen == gen
	t.mu.Unlock()
	if current {
		t.Hide()
	}
}

// Hide takes the window down now.
func (t *Transient) Hide() {
	t.mu.Lock()
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	was := t.shown
	t.shown = false
	t.mu.Unlock()
	if was {
		t.winMu.Lock()
		t.win.win.Hide()
		t.winMu.Unlock()
	}
}

// Shown reports whether the window is up.
func (t *Transient) Shown() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.shown
}
