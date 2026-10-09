package wizard

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/logpane"
	"github.com/ushineko/fynedesygn/steps"
	"github.com/ushineko/fynedesygn/widgets"
)

// Job is the work a progress page runs. It reports through r, and returns
// an error wrapping context.Canceled when it stopped because ctx was
// cancelled.
type Job func(ctx context.Context, r *Reporter) error

// ProgressPage runs a job with a step list beside its log (R4). It starts
// the job the first time it becomes current, and it is the point of no
// return: once the job has started, Back is gone for good.
type ProgressPage struct {
	title string
	job   Job
	done  string
	list  *steps.List
	pane  *logpane.Pane
	ran   bool
	// count is the bar region; nil without WithBar.
	count *counter
}

// WithBar adds a determinate bar, a status line and the current item under
// the step list and the log (spec 063). The job reports to it through
// Reporter.Progress. Call it before the wizard is built.
func (p *ProgressPage) WithBar() *ProgressPage {
	p.count = newCounter()
	return p
}

// Progress makes a progress page whose job has the named steps. done is
// the message line's text when the job succeeds.
func Progress(title string, stepNames []string, done string, job Job) *ProgressPage {
	return &ProgressPage{
		title: title, job: job, done: done,
		list: steps.New(stepNames...),
		pane: logpane.New(logpane.NewModel(logpane.DefaultMaxLines)),
	}
}

// Title implements Page.
func (p *ProgressPage) Title() string { return p.title }

// PointOfNoReturn implements PointOfNoReturn.
func (p *ProgressPage) PointOfNoReturn() bool { return true }

// Log is the page's log, for a test or a program that keeps it.
func (p *ProgressPage) Log() *logpane.Model { return p.pane.Model() }

// Steps is the page's step list.
func (p *ProgressPage) Steps() *steps.List { return p.list }

// Build implements Page.
func (p *ProgressPage) Build(w *Wizard) fyne.CanvasObject {
	height := float32(300)
	var bottom fyne.CanvasObject
	if p.count != nil {
		bottom = p.count.widget()
		// The log gives up the bar's height, so the page is no taller with
		// the bar than without it.
		height -= bottom.MinSize().Height
	}
	return container.NewBorder(nil, bottom,
		widgets.FixedWidth(container.NewVBox(
			widget.NewLabelWithStyle("Steps", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			p.list.Widget()), 220),
		nil,
		p.pane.Widget(logpane.Options{Title: "Output", Height: height, Clipboard: w.App.Clipboard()}))
}

// Enter implements Enterer: the first time, it starts the job.
func (p *ProgressPage) Enter(w *Wizard) {
	if p.ran {
		return
	}
	p.ran = true
	r := &Reporter{w: w, p: p}
	w.startJob(func(ctx context.Context) error {
		if w.Window != nil {
			stop := p.pane.Pump()
			defer stop()
			if p.count != nil {
				stopCount := p.count.pump(w.do)
				defer stopCount()
			}
		}
		return p.job(ctx, r)
	})
}

// Reporter is how a job reports progress. Its methods are safe to call from
// the job's goroutine (R9).
type Reporter struct {
	w *Wizard
	p *ProgressPage
}

// Advance marks step i running with a note, and earlier steps done.
func (r *Reporter) Advance(i int, note string) {
	r.w.do(func() { r.p.list.Advance(i, note) })
}

// Finish marks step i done with a note.
func (r *Reporter) Finish(i int, note string) {
	r.w.do(func() { r.p.list.Finish(i, note) })
}

// Progress sets the bar to fraction (0 to 1), the status line, and the item
// line (spec 063). It is safe to call from the job's goroutine as often as
// once per file: the page draws the latest values on a timer. It does
// nothing on a page without WithBar.
func (r *Reporter) Progress(fraction float64, status, item string) {
	if r.p.count == nil {
		return
	}
	r.p.count.set(fraction, status, item)
	if r.w.Window == nil {
		r.p.count.draw()
	}
}

// Log adds a line to the log.
func (r *Reporter) Log(level logpane.Level, text string) {
	r.p.pane.Log(level, text)
}
