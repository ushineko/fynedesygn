/*
Package steps is the step list a job shows beside its log: one row per
step with a marker, the name and a note, updated in place as the job
advances. Ported from nmsbonker's buildRun steps (same author, MIT).

A step list is for jobs with real steps. A job without them shows the
indeterminate busy indicator instead: a bar that filled steadily would be
inventing a number.
*/
package steps

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/widgets"
)

// State is where a step is.
type State int

// The states, in the order a step moves through them.
const (
	Pending State = iota
	Running
	Done
	Failed
	Cancelled
)

// String names the state for a note or a test.
func (s State) String() string {
	switch s {
	case Running:
		return "running"
	case Done:
		return "done"
	case Failed:
		return "failed"
	case Cancelled:
		return "cancelled"
	case Pending:
		return "pending"
	}
	return "pending"
}

// Status is how the row is painted.
func (s State) Status() fd.Status {
	switch s {
	case Done:
		return fd.StatusGood
	case Failed:
		return fd.StatusBad
	case Cancelled:
		return fd.StatusWarn
	case Pending, Running:
		return fd.StatusInfo
	}
	return fd.StatusInfo
}

// Icon is the row's marker resource.
func (s State) Icon() fyne.Resource {
	switch s {
	case Running:
		return fynetheme.MediaPlayIcon()
	case Done:
		return fynetheme.ConfirmIcon()
	case Failed:
		return fynetheme.ErrorIcon()
	case Cancelled:
		return fynetheme.CancelIcon()
	case Pending:
		return fynetheme.RadioButtonIcon()
	}
	return fynetheme.RadioButtonIcon()
}

// Step is one row.
type Step struct {
	Name  string
	State State
	Note  string
}

// row is the live widgets of one step, nil when detached.
type row struct {
	icon *widget.Icon
	name *widget.Label
	note *widget.Label
}

// List holds the steps and, while a section shows it, their rows. All
// methods are for the UI thread: a worker hops with fyne.Do first.
type List struct {
	steps []Step
	rows  []row
}

// New makes a list of pending steps.
func New(names ...string) *List {
	l := &List{steps: make([]Step, len(names))}
	for i, n := range names {
		l.steps[i] = Step{Name: n}
	}
	return l
}

// Steps is a copy of the current state.
func (l *List) Steps() []Step { return append([]Step(nil), l.steps...) }

// Set changes one step and repaints its row.
func (l *List) Set(i int, st State, note string) {
	if i < 0 || i >= len(l.steps) {
		return
	}
	l.steps[i].State, l.steps[i].Note = st, note
	l.paint(i)
}

// Advance marks step i running and every earlier step that is not finished
// done, for a job that reports where it is rather than what it finished.
func (l *List) Advance(i int, note string) {
	for j := 0; j < i && j < len(l.steps); j++ {
		if l.steps[j].State == Pending || l.steps[j].State == Running {
			l.Set(j, Done, l.steps[j].Note)
		}
	}
	l.Set(i, Running, note)
}

// Finish marks step i done with a note.
func (l *List) Finish(i int, note string) { l.Set(i, Done, note) }

// Stop marks every running step st (Failed or Cancelled) with the note, and
// leaves pending steps pending: they never started.
func (l *List) Stop(st State, note string) {
	for i, s := range l.steps {
		if s.State == Running {
			l.Set(i, st, note)
		}
	}
}

// Reset puts every step back to pending with no note.
func (l *List) Reset() {
	for i := range l.steps {
		l.Set(i, Pending, "")
	}
}

// Widget builds the rows. Call again after Detach when the section rebuilds.
func (l *List) Widget() fyne.CanvasObject {
	l.rows = make([]row, len(l.steps))
	objs := make([]fyne.CanvasObject, 0, len(l.steps))
	for i := range l.steps {
		r := row{
			icon: widget.NewIcon(Pending.Icon()),
			name: widget.NewLabel(""),
			note: widget.NewLabel(""),
		}
		r.note.Importance = widget.LowImportance
		r.note.Truncation = fyne.TextTruncateEllipsis
		l.rows[i] = r
		objs = append(objs, container.NewBorder(nil, nil,
			container.NewHBox(r.icon, r.name), nil, r.note))
		l.paint(i)
	}
	return container.NewVBox(objs...)
}

// Detach forgets the rows; Set keeps updating the state for the next Widget.
func (l *List) Detach() { l.rows = nil }

// paint brings row i up to date with step i.
func (l *List) paint(i int) {
	if i >= len(l.rows) {
		return
	}
	r, s := l.rows[i], l.steps[i]
	r.icon.SetResource(s.State.Icon())
	r.name.Importance = widgets.ImportanceFor(s.State.Status())
	if s.State == Running {
		r.name.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		r.name.TextStyle = fyne.TextStyle{}
	}
	r.name.SetText(s.Name)
	r.note.SetText(s.Note)
}
