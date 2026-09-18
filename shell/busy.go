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
	return s.busy(what, nil)
}

/*
BusyCancellable is Busy with a Cancel button on the popup, which calls cancel.

For a job that holds the indicator itself rather than running through
PerformCancellable: one that pumps a log around the call, keeps its own record
of whether it was cancelled, or reports its own outcome. The popup is modal, so
while it is up the Cancel on it is the only Cancel the user can reach; a
toolbar button left enabled behind it cannot be clicked.

The button goes when the operation does. The cancel is the caller's own, so it
may be a context's CancelFunc or a method that does the program's cancelling
bookkeeping as well.
*/
func (s *Shell) BusyCancellable(what string, cancel context.CancelFunc) func() {
	return s.busy(what, cancel)
}

// busy is Busy and BusyCancellable: cancel nil means no button.
func (s *Shell) busy(what string, cancel context.CancelFunc) func() {
	job := &busyJob{cancel: cancel}
	fyne.Do(func() {
		s.busyMu.Lock()
		s.busyCount++
		s.busyWhat = what
		first := s.busyCount == 1
		if cancel != nil {
			s.busyCancel = job
		}
		s.showBusy()
		s.busyMu.Unlock()
		// regate outside the lock: it rebuilds the section, and a builder
		// calling Working() would meet a mutex this goroutine already holds.
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
				// This operation's cancel goes when this operation does, even
				// if a load beside it is still holding the popup up: a Cancel
				// button that outlives the job it cancels does nothing when
				// pressed. Compared by pointer because func values are not
				// comparable, and a second cancellable operation may have
				// replaced this one.
				if s.busyCancel == job {
					s.busyCancel = nil
				}
				if last {
					s.hideBusy()
				} else {
					s.showBusy()
				}
				s.busyMu.Unlock()
				if last {
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

/*
showBusy puts the progress popup up, centred and modal, once the operation has
lasted long enough to deserve one.

Modal on purpose: the window runs one operation at a time and every button that
starts work is disabled while one runs, so a popup that also swallows clicks
changes nothing about what can be done, and says plainly why the window is not
answering. The exception is a cancellable job, which leaves its Cancel enabled
— and the modal would swallow that click too. So the Cancel comes onto the
popup, where it is the one control in front of the user.

Called again while the popup is up: the caption is updated, and the popup is
rebuilt if a cancel has appeared or gone, so the button tracks the operation
rather than the moment the popup happened to be built.

Call with busyMu held. The popup is written here, on the delay timer's
goroutine, and read by the next caller of Busy on theirs; Fyne's real driver
serialises both onto the main loop but its test driver runs fyne.Do inline on
whichever goroutine called (quirk 11), so the lock is what makes the busy
fields safe rather than the toolkit.
*/
func (s *Shell) showBusy() {
	if !s.OnScreen() {
		return
	}
	if s.busyPop != nil {
		s.busyLabel.SetText(s.busyWhat)
		if (s.busyCancel != nil) == (s.busyCancelBtn != nil) {
			return
		}
		// Rebuilt rather than re-laid out: a modal popup sizes and centres
		// itself on the content it was made with, and this happens at most
		// twice in an operation.
		s.busyPop.Hide()
		s.busyPop = s.busyPopUp(s.busyWhat)
		s.busyPop.Show()
		return
	}
	s.busySeq++
	seq := s.busySeq
	go func() {
		time.Sleep(BusyDelay)
		fyne.Do(func() {
			s.busyMu.Lock()
			defer s.busyMu.Unlock()
			if s.busySeq != seq || s.busyCount == 0 || s.busyPop != nil {
				return
			}
			s.busyPop = s.busyPopUp(s.busyWhat)
			s.busyPop.Show()
		})
	}()
}

// busyPopup returns the popup that is up, if any. The busy fields are behind
// busyMu, so a reader not already holding it comes through here.
func (s *Shell) busyPopup() *widget.PopUp {
	s.busyMu.Lock()
	defer s.busyMu.Unlock()
	return s.busyPop
}

// busyPopUp builds the popup for the caption, with a Cancel button when a
// cancellable operation is holding it. Call with busyMu held.
func (s *Shell) busyPopUp(what string) *widget.PopUp {
	s.busyLabel = widget.NewLabel(what)
	s.busyLabel.Alignment = fyne.TextAlignCenter
	bar := widget.NewProgressBarInfinite()
	items := []fyne.CanvasObject{s.busyLabel, widgets.FixedWidth(bar, busyBarWidth)}

	s.busyCancelBtn = nil
	if job := s.busyCancel; job != nil {
		btn := widget.NewButton("Cancel", nil)
		btn.OnTapped = func() {
			// Dead the moment it has been pressed. Cancelling is rarely
			// instant — processes have to be stopped, a previous output put
			// back — and a button that still looks live is one pressed again.
			btn.Disable()
			job.cancel()
		}
		s.busyCancelBtn = btn
		items = append(items, container.NewCenter(btn))
	}
	return widget.NewModalPopUp(container.NewPadded(container.NewVBox(items...)), s.Window.Canvas())
}

// hideBusy takes the progress popup down. Call with busyMu held.
func (s *Shell) hideBusy() {
	s.busySeq++ // a pending showBusy timer finds a different sequence and stops
	if s.busyPop != nil {
		s.busyPop.Hide()
		s.busyPop, s.busyLabel = nil, nil
	}
	s.busyCancel, s.busyCancelBtn = nil, nil
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
	// The popup's Cancel is the operation's own context cancel; a
	// non-cancellable one hands busy no cancel and gets no button.
	var button context.CancelFunc
	if cancellable {
		button = cancel
	}
	go func() {
		defer cancel()
		done := s.busy(what, button)
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

/*
Load runs a section's data load off the UI thread with the busy indicator
up. Unlike Perform it does not refuse while something else is working: a
section may start several loads, and a load may run beside an operation.
With no window (a headless test) the function runs inline, for the same
reason Perform does. The loader sets its loaded flag before calling Load so
a rebuild mid-load does not start a second one, and hops back with fyne.Do
to store the result and Refresh.
*/
func (s *Shell) Load(what string, fn func(ctx context.Context) error) {
	if !s.OnScreen() {
		if err := fn(context.Background()); err != nil {
			s.Report(what, err)
		}
		return
	}
	go func() {
		done := s.Busy(what)
		defer done()
		if err := fn(context.Background()); err != nil {
			fyne.Do(func() { s.Report(what, err) })
		}
	}()
}

// busyJob is one cancellable operation's claim on the popup's Cancel button.
// A pointer, so the operation that set it can tell when it finishes whether
// the button is still its own: func values are not comparable, and a second
// operation may have taken the popup over.
type busyJob struct{ cancel context.CancelFunc }
