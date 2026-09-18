package shell

import (
	"context"
	"errors"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/widgets"
)

// BusyDelay is how long an operation runs before the progress popup appears.
// Most operations finish inside it, and a popup that blinks for a tenth of a
// second on every click is worse than none.
const BusyDelay = 300 * time.Millisecond

// busyBarWidth is the infinite progress bar's width in the popup.
const busyBarWidth float32 = 320

/*
Busy shows an indeterminate progress indicator until the returned function is
called. The returned function may be called more than once; only the first
call counts.

Indeterminate on purpose: a directory scan does not know how many files it
will walk until it has walked them, and a network round trip is one long step
rather than many short ones. A bar that filled steadily would be inventing a
number. A job with real steps shows its own step list.

Safe to call from a goroutine: it hops to the UI thread itself, and so does
the function it returns. Callers are core operations running off the UI
thread, so requiring them to marshal by hand would be an invitation to forget.

Every core call gets one, not just the obviously slow ones: the machine this
runs on is not the machine it was written on. A window that sits still with no
explanation reads as frozen, and the button that looks like it did nothing is
the button that gets clicked twice.
*/
func (s *Shell) Busy(what string) func() {
	fyne.Do(func() {
		s.busyMu.Lock()
		s.busyCount++
		s.busyWhat = what
		first := s.busyCount == 1
		s.busyMu.Unlock()
		s.showBusy()
		if first {
			s.regate()
		}
	})

	var once sync.Once
	return func() {
		once.Do(func() {
			fyne.Do(func() {
				s.busyMu.Lock()
				s.busyCount--
				last := s.busyCount <= 0
				if last {
					s.busyCount, s.busyWhat = 0, ""
				}
				s.busyMu.Unlock()
				if last {
					s.hideBusy()
					s.regate()
				}
			})
		})
	}
}

// Working reports whether an operation is in flight: a Busy count above zero
// or the program's AlsoWorking predicate. One operation at a time, with the
// other buttons disabled rather than hidden, so the window does not change
// shape as work starts and finishes.
func (s *Shell) Working() bool {
	s.busyMu.Lock()
	n := s.busyCount
	s.busyMu.Unlock()
	if n > 0 {
		return true
	}
	return s.opts.AlsoWorking != nil && s.opts.AlsoWorking()
}

// Gate disables the buttons that start work while something is running. A
// builder calls it on the buttons it just made.
func (s *Shell) Gate(buttons ...*widget.Button) {
	if !s.Working() {
		return
	}
	for _, b := range buttons {
		b.Disable()
	}
}

/*
regate rebuilds the current section when work starts and when it stops.

Every section disables the buttons that start work while something is
running, and it does that as it is built, so a section built while an
operation was in flight comes out with dead buttons and nothing turns them
back on. The loads a window starts at launch are still running when the first
section is drawn, so without this the first toolbar came up permanently
disabled and the window looked finished while refusing to do anything.

Rebuilding from state is the fix rather than walking a list of buttons,
because "disabled" has several causes at once and only the builder knows all
of them.
*/
func (s *Shell) regate() {
	s.Refresh()
	s.RedrawStatus()
}

// showBusy puts the progress popup up, centred and modal, once the operation
// has lasted long enough to deserve one. Modal on purpose: the window runs one
// operation at a time and every button is disabled while one runs, so a popup
// that also swallows clicks changes nothing about what can be done, and says
// plainly why the window is not answering.
func (s *Shell) showBusy() {
	if !s.OnScreen() {
		return
	}
	if s.busyPop != nil {
		s.busyLabel.SetText(s.busyWhat)
		return
	}
	s.busySeq++
	seq := s.busySeq
	go func() {
		time.Sleep(BusyDelay)
		fyne.Do(func() {
			s.busyMu.Lock()
			count, what := s.busyCount, s.busyWhat
			s.busyMu.Unlock()
			if s.busySeq != seq || count == 0 || s.busyPop != nil {
				return
			}
			s.busyLabel = widget.NewLabel(what)
			s.busyLabel.Alignment = fyne.TextAlignCenter
			bar := widget.NewProgressBarInfinite()
			items := []fyne.CanvasObject{s.busyLabel, widgets.FixedWidth(bar, busyBarWidth)}
			if s.busyCancel != nil {
				cancel := widget.NewButton("Cancel", func() {
					if s.busyCancel != nil {
						s.busyCancel()
					}
				})
				items = append(items, container.NewCenter(cancel))
			}
			s.busyPop = widget.NewModalPopUp(container.NewPadded(container.NewVBox(items...)), s.Window.Canvas())
			s.busyPop.Show()
		})
	}()
}

// hideBusy takes the progress popup down.
func (s *Shell) hideBusy() {
	s.busySeq++ // a pending showBusy timer finds a different sequence and stops
	if s.busyPop != nil {
		s.busyPop.Hide()
		s.busyPop, s.busyLabel = nil, nil
	}
	s.busyCancel = nil
}

/*
Perform runs one operation off the UI thread with the busy indicator up.

what is the popup's caption, so it is a phrase in the present participle:
"Clearing the history...", not "clear". Every core call from a window goes
through Perform or through a loader that calls Busy; a raw goroutine reaching
into core would be a window that sits still with no explanation.

A second request while one runs is refused with a warning banner. With no
window (a headless test) the function runs inline: Fyne's test driver runs
fyne.Do on the calling goroutine rather than serialising onto a main loop, so
a worker refreshing a widget would genuinely race the test driving it, and a
test wants a finished operation when the button returns rather than one that
lands soon.
*/
func (s *Shell) Perform(what string, fn func(ctx context.Context) error) {
	s.perform(what, false, fn)
}

// PerformCancellable is Perform with a Cancel button on the busy popup; the
// button cancels the context handed to fn.
func (s *Shell) PerformCancellable(what string, fn func(ctx context.Context) error) {
	s.perform(what, true, fn)
}

func (s *Shell) perform(what string, cancellable bool, fn func(ctx context.Context) error) {
	if s.Working() {
		s.Flash("Something is already running. Wait for it to finish, or cancel it.", fd.StatusWarn)
		return
	}
	if !s.OnScreen() {
		if err := fn(context.Background()); err != nil {
			s.Report(what, err)
		}
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	if cancellable {
		s.busyCancel = cancel
	}
	go func() {
		defer cancel()
		done := s.Busy(what)
		defer done()
		if err := fn(ctx); err != nil {
			fyne.Do(func() { s.Report(what, err) })
		}
	}()
}

// Report shows an operation's failure as a banner that stays until dismissed.
// Cancellation is not a failure and shows nothing. Call on the UI thread.
func (s *Shell) Report(what string, err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	s.Flash(what+" failed: "+err.Error(), fd.StatusBad)
}

// OK reports success and invalidates, so every section fetches what the
// operation changed. Call on the UI thread.
func (s *Shell) OK(msg string) {
	s.Flash(msg, fd.StatusGood)
	s.Invalidate()
}
