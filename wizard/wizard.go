/*
Package wizard is the third window archetype: a fixed sequence of pages with
Back, Next and Cancel, where the user cannot go on until the current page is
complete, and where the last pages run a job and report its result. An
installer is the first program built on it (fynstall); a first-run setup or
an import has the same shape.

A shell lets the user go to any section in any order, and a glance window is
read rather than operated. A wizard has an order, a current page, and a point
after which Back no longer means anything. See docs/wizard.md.

The window is laid out as

	Border{
	  left:   the page list, in steps vocabulary
	  center: Border{ top: the page heading, center: the one scroller }
	  bottom: Separator, then Border{ left: the message line,
	                                  right: Back, Next, Cancel }
	}

The message line is always there and the buttons have fixed widths, so a
message, a new button label or a running job never moves the interface.
*/
package wizard

import (
	"context"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/dialogs"
	"github.com/ushineko/fynedesygn/steps"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// Page is one step of a wizard. Optional behaviour is a separate interface
// (Validator, Enterer, Leaver, Skipper, Labeller, PointOfNoReturn,
// Changer), the way shell sections opt into Arriver.
//
// Pages are made before Run creates the app, so a page's constructor holds
// data and only Build touches widgets: Fyne cannot refresh a widget before
// there is an app.
type Page interface {
	// Title names the page in the page list and heads it.
	Title() string
	// Build makes the page's content. It is called once, the first time
	// the page becomes current; the page updates what it built in place.
	Build(w *Wizard) fyne.CanvasObject
}

// Validator is a page that can be incomplete. Next is enabled only while
// Valid is true; the page calls Wizard.Revalidate when its state changes.
type Validator interface {
	Valid() bool
}

// Enterer is a page that does something each time it becomes current, such
// as a summary reading the choices made on earlier pages.
type Enterer interface {
	Enter(w *Wizard)
}

// Leaver is a page that is asked before the wizard leaves it. A non-nil
// error keeps the page current and shows on the message line.
type Leaver interface {
	Leave(w *Wizard, forward bool) error
}

// Skipper is a page that may not apply. Back and Next pass over it while
// Skip is true, and the page list marks it skipped.
type Skipper interface {
	Skip() bool
}

// Labeller is a page that names its Next button, such as "Install" on a
// summary. The last page's is "Finish" unless it says otherwise.
type Labeller interface {
	NextLabel() string
}

// PointOfNoReturn is a page after which Back is never available again,
// normally the page that runs the job.
type PointOfNoReturn interface {
	PointOfNoReturn() bool
}

// Changer is a page that holds input the user would lose. Cancel asks for
// confirmation while any page that has been shown reports a change.
type Changer interface {
	Changed() bool
}

// Outcome is how a wizard ended.
type Outcome int

const (
	// Cancelled is a wizard the user left before the work was done, or
	// whose job the user stopped.
	Cancelled Outcome = iota
	// Finished is a wizard the user went through to the end.
	Finished
	// Failed is a wizard whose job failed, closed by the user afterwards.
	Failed
)

func (o Outcome) String() string {
	switch o {
	case Finished:
		return "finished"
	case Failed:
		return "failed"
	default:
		return "cancelled"
	}
}

// Result is what Run returns once the window has closed.
type Result struct {
	Outcome Outcome
	// Err is the job's error when Outcome is Failed.
	Err error
	// Checks are the finish page's checks, by their text.
	Checks map[string]bool
}

// Options describe a wizard.
type Options struct {
	// AppID names the app, and on Wayland the window's app_id. Used by New.
	AppID string
	// Name is the window title, and heads the page list.
	Name string
	// Icon is the window icon; nil leaves Fyne's.
	Icon fyne.Resource
	// Size is the window size; zero means DefaultWidth x DefaultHeight.
	Size fyne.Size
	// Appearance is applied to the app by New; nil means the platform
	// default. NewIn leaves the app's theme alone.
	Appearance *fdtheme.Appearance
	// Pages in order. At least one.
	Pages []Page
}

// The default window size.
const (
	DefaultWidth  float32 = 820
	DefaultHeight float32 = 560
	// ListWidth is the width of the page list.
	ListWidth float32 = 200
)

// Labels on the buttons. Next's width is the widest of its possible labels,
// so the row never changes width.
const (
	LabelBack   = "Back"
	LabelNext   = "Next"
	LabelFinish = "Finish"
	LabelCancel = "Cancel"
	LabelClose  = "Close"
)

// Wizard is a running wizard: its window, its pages and where it is.
type Wizard struct {
	// App is the app the wizard runs in.
	App fyne.App
	// Window is nil for a Headless wizard.
	Window fyne.Window

	opts    Options
	cur     int
	shown   []bool
	content []fyne.CanvasObject
	// noReturn is the index of the point-of-no-return page once it has
	// been entered, else -1.
	noReturn int
	// job state, set by the progress page.
	running   bool
	succeeded bool
	failed    error
	cancelled bool
	jobCancel context.CancelFunc
	// closing is true once the user asked to leave while a job ran; the
	// window closes when the job returns.
	closing bool
	closed  bool
	result  Result

	list    *steps.List
	title   *widget.Label
	scroll  *container.Scroll
	message *widget.Label
	back    *widget.Button
	next    *widget.Button
	cancel  *widget.Button
	buttons *fyne.Container
	root    fyne.CanvasObject

	// confirm asks before something is lost; tests replace it.
	confirm func(title, detail, ok string, do func())
	// do runs fn on the UI thread: fyne.Do with a window, inline without.
	do func(fn func())
}

// Run makes the wizard, shows it and blocks until its window closes.
func Run(o Options) Result {
	w := New(o)
	w.Window.ShowAndRun()
	return w.Result()
}

// New creates the app, applies the appearance, and builds the window
// without showing it.
func New(o Options) *Wizard {
	// Before the toolkit starts: GLFW reads the cursor theme at init.
	fdtheme.ApplyCursorTheme()
	a := app.NewWithID(o.AppID)
	ap := fdtheme.DefaultAppearance()
	if o.Appearance != nil {
		ap = *o.Appearance
	}
	ap.Apply(a)
	if o.Icon != nil {
		a.SetIcon(o.Icon)
	}
	w := NewIn(a, o)
	w.Window.SetMaster()
	return w
}

/*
NewIn builds the wizard's window over an app the caller already has, such as
the gallery's, and leaves the app's theme alone. The window is not shown.
*/
func NewIn(a fyne.App, o Options) *Wizard {
	w := newWizard(a, o)
	w.Window = a.NewWindow(o.Name)
	if o.Icon != nil {
		w.Window.SetIcon(o.Icon)
	}
	w.do = fyne.Do
	w.confirm = func(title, detail, ok string, do func()) {
		dialogs.ConfirmDestructive(w.Window, title, detail, ok, do)
	}
	size := o.Size
	if size.IsZero() {
		size = fyne.NewSize(DefaultWidth, DefaultHeight)
	}
	w.Window.Resize(size)
	w.Window.SetFixedSize(true)
	w.Window.SetContent(w.root)
	// The title bar's close is Cancel: it asks the same questions and
	// waits for a running job the same way.
	w.Window.SetCloseIntercept(w.Cancel)
	return w.start()
}

// Headless builds a wizard with no window over an existing app (a test
// app). A job runs inline and Cancel's confirmation is accepted at once, so
// a test drives it deterministically.
func Headless(a fyne.App, o Options) *Wizard {
	w := newWizard(a, o)
	w.do = func(fn func()) { fn() }
	w.confirm = func(_, _, _ string, do func()) { do() }
	return w.start()
}

func newWizard(a fyne.App, o Options) *Wizard {
	if len(o.Pages) == 0 {
		panic("wizard: Options.Pages is empty")
	}
	w := &Wizard{
		App: a, opts: o, noReturn: -1,
		shown:   make([]bool, len(o.Pages)),
		content: make([]fyne.CanvasObject, len(o.Pages)),
	}
	titles := make([]string, len(o.Pages))
	for i, p := range o.Pages {
		titles[i] = p.Title()
	}
	w.list = steps.New(titles...)
	// A bold title over a rule, as widgets.Card heads a card. Not
	// widgets.Heading, which keeps a row for its blurb even when empty.
	w.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// Not nil: NewIn sets the window content before the first page is
	// shown, and a scroller measures its content when it is laid out.
	w.scroll = container.NewVScroll(container.NewStack())
	w.message = widget.NewLabel("")
	w.message.Truncation = fyne.TextTruncateEllipsis
	w.back = widget.NewButton(LabelBack, w.Back)
	w.next = widget.NewButton(LabelNext, w.Next)
	w.next.Importance = widget.HighImportance
	w.cancel = widget.NewButton(LabelCancel, w.Cancel)

	nextLabels := []string{LabelNext, LabelFinish}
	for _, p := range o.Pages {
		if l, ok := p.(Labeller); ok {
			nextLabels = append(nextLabels, l.NextLabel())
		}
	}
	w.buttons = container.NewHBox(
		widgets.FixedWidth(w.back, widest(LabelBack)),
		widgets.FixedWidth(w.next, widest(nextLabels...)),
		widgets.FixedWidth(w.cancel, widest(LabelCancel, LabelClose)),
	)
	row := container.NewBorder(nil, nil, nil, w.buttons, w.message)
	side := container.NewBorder(
		widget.NewLabelWithStyle(o.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil, w.list.Widget())
	w.root = container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), container.NewPadded(row)),
		container.NewHBox(widgets.FixedWidth(container.NewPadded(side), ListWidth), widget.NewSeparator()),
		nil,
		container.NewBorder(container.NewVBox(w.title, widget.NewSeparator()), nil, nil, nil, w.scroll),
	)
	return w
}

// start shows the first page. It runs after the constructor has set do and
// confirm, because the first page may be one that starts a job.
func (w *Wizard) start() *Wizard {
	w.show(w.step(-1, +1))
	return w
}

// widest is the width a button needs for the widest of labels.
func widest(labels ...string) float32 {
	var most float32
	for _, l := range labels {
		most = max(most, widget.NewButton(l, nil).MinSize().Width)
	}
	return most
}

// Content is the window's content: the whole wizard. A Headless wizard's
// test drives it through this.
func (w *Wizard) Content() fyne.CanvasObject { return w.root }

// Current is the page being shown.
func (w *Wizard) Current() Page { return w.opts.Pages[w.cur] }

// Message is the text on the message line.
func (w *Wizard) Message() string { return w.message.Text }

// Result is how the wizard ended; Outcome is Cancelled until it has.
func (w *Wizard) Result() Result { return w.result }

// SetMessage shows text on the message line in the colour of st; "" clears
// it.
func (w *Wizard) SetMessage(text string, st fd.Status) {
	w.message.Importance = widgets.ImportanceFor(st)
	w.message.SetText(text)
}

// Revalidate updates the buttons after a page's state changes.
func (w *Wizard) Revalidate() { w.update() }

// Next leaves the current page forward, or finishes on the last page.
func (w *Wizard) Next() {
	if w.next.Disabled() {
		return
	}
	if l, ok := w.Current().(Leaver); ok {
		if err := l.Leave(w, true); err != nil {
			w.SetMessage(err.Error(), fd.StatusBad)
			return
		}
	}
	i := w.step(w.cur, +1)
	if i < 0 {
		w.finish(Finished, nil)
		return
	}
	w.show(i)
}

// Back returns to the previous page that applies.
func (w *Wizard) Back() {
	if w.back.Disabled() {
		return
	}
	if l, ok := w.Current().(Leaver); ok {
		if err := l.Leave(w, false); err != nil {
			w.SetMessage(err.Error(), fd.StatusBad)
			return
		}
	}
	if i := w.step(w.cur, -1); i >= 0 {
		w.show(i)
	}
}

/*
Cancel is the Cancel button and the title bar's close.

Before the point of no return it closes the window, after a confirmation if a
page reports a change. While a job runs it asks, cancels the job, and closes
when the job returns. After a failure it is Close. After the job succeeded,
closing is finishing.
*/
func (w *Wizard) Cancel() {
	switch {
	case w.closed:
	case w.running:
		w.confirm("Stop "+w.opts.Name+"?",
			"The work in progress stops and is undone where it can be. The window closes when it has stopped.",
			"Stop", func() {
				w.closing = true
				w.cancelJob()
			})
	case w.failed != nil:
		w.finish(Failed, w.failed)
	case w.succeeded:
		w.finish(Finished, nil)
	case w.changed():
		w.confirm("Leave "+w.opts.Name+"?", "Nothing has been changed yet, and the choices made here are not kept.",
			"Leave", func() { w.finish(Cancelled, nil) })
	default:
		w.finish(Cancelled, nil)
	}
}

func (w *Wizard) changed() bool {
	for i, p := range w.opts.Pages {
		if c, ok := p.(Changer); ok && w.shown[i] && c.Changed() {
			return true
		}
	}
	return false
}

func (w *Wizard) finish(o Outcome, err error) {
	if w.closed {
		return
	}
	w.closed = true
	w.result = Result{Outcome: o, Err: err, Checks: map[string]bool{}}
	for _, p := range w.opts.Pages {
		if f, ok := p.(*FinishPage); ok {
			for _, c := range f.checks {
				w.result.Checks[c.Text] = c.Checked
			}
		}
	}
	if w.Window != nil {
		w.Window.Close()
	}
}

// step finds the next page from i in direction dir that is not skipped,
// or -1.
func (w *Wizard) step(i, dir int) int {
	for i += dir; i >= 0 && i < len(w.opts.Pages); i += dir {
		if s, ok := w.opts.Pages[i].(Skipper); !ok || !s.Skip() {
			return i
		}
	}
	return -1
}

func (w *Wizard) show(i int) {
	w.cur = i
	p := w.opts.Pages[i]
	if w.content[i] == nil {
		w.content[i] = p.Build(w)
	}
	w.shown[i] = true
	if n, ok := p.(PointOfNoReturn); ok && n.PointOfNoReturn() && w.noReturn < 0 {
		w.noReturn = i
	}
	w.title.SetText(p.Title())
	w.scroll.Content = container.NewPadded(w.content[i])
	w.scroll.Refresh()
	w.scroll.ScrollToTop()
	w.SetMessage("", fd.StatusInfo)
	if e, ok := p.(Enterer); ok {
		e.Enter(w)
	}
	w.update()
}

// update sets the page list and the buttons from the state.
func (w *Wizard) update() {
	for i, p := range w.opts.Pages {
		switch s, skip := p.(Skipper); {
		case skip && s.Skip():
			w.list.Set(i, steps.Pending, "skipped")
		case i == w.cur:
			w.list.Set(i, steps.Running, "")
		case i < w.cur:
			w.list.Set(i, steps.Done, "")
		default:
			w.list.Set(i, steps.Pending, "")
		}
	}
	if w.failed != nil {
		w.list.Set(w.cur, steps.Failed, "")
	}

	p := w.Current()
	last := w.step(w.cur, +1) < 0
	first := w.step(w.cur, -1) < 0
	valid := true
	if v, ok := p.(Validator); ok {
		valid = v.Valid()
	}
	setEnabled(w.back, !first && !w.running && w.noReturn < 0)
	nextOK := valid && !w.running && w.failed == nil
	if w.cur == w.noReturn && !w.succeeded {
		nextOK = false
	}
	setEnabled(w.next, nextOK)
	label := LabelNext
	if last {
		label = LabelFinish
	}
	if l, ok := p.(Labeller); ok {
		label = l.NextLabel()
	}
	w.next.SetText(label)

	switch {
	case w.failed != nil:
		w.cancel.SetText(LabelClose)
		w.cancel.Enable()
	case w.noReturn >= 0 && !w.running:
		// Past the point of no return there is nothing to cancel: the
		// title bar still closes, and that is finishing.
		w.cancel.SetText(LabelCancel)
		w.cancel.Disable()
	default:
		w.cancel.SetText(LabelCancel)
		w.cancel.Enable()
	}
}

func setEnabled(b *widget.Button, on bool) {
	if on {
		b.Enable()
	} else {
		b.Disable()
	}
}

// startJob runs job for the progress page p. With a window it runs on a
// goroutine and reports through fyne.Do; headless it runs inline.
func (w *Wizard) startJob(job func(ctx context.Context) error) {
	ctx, cancel := context.WithCancel(context.Background())
	w.jobCancel = cancel
	w.running = true
	w.update()
	run := func() {
		// The job's own error decides. A job that finished its work as the
		// cancel arrived returns nil, and its work is done: reporting it as
		// cancelled would tell the user an install had not happened when it
		// had.
		err := job(ctx)
		w.do(func() { w.jobDone(err, errors.Is(err, context.Canceled)) })
		cancel()
	}
	if w.Window == nil {
		run()
		return
	}
	go run()
}

func (w *Wizard) cancelJob() {
	if w.jobCancel != nil {
		w.jobCancel()
	}
}

func (w *Wizard) jobDone(err error, cancelled bool) {
	w.running = false
	pp, _ := w.Current().(*ProgressPage)
	switch {
	case cancelled:
		w.cancelled = true
		if pp != nil {
			pp.list.Stop(steps.Cancelled, "stopped")
		}
		w.SetMessage("Stopped.", fd.StatusWarn)
		w.update()
		w.finish(Cancelled, nil)
		return
	case err != nil:
		w.failed = err
		if pp != nil {
			pp.list.Stop(steps.Failed, "failed")
		}
		w.SetMessage(fmt.Sprintf("Failed: %v", err), fd.StatusBad)
	default:
		w.succeeded = true
		if pp != nil {
			w.SetMessage(pp.done, fd.StatusGood)
		}
	}
	w.update()
	if w.closing {
		w.Cancel()
	}
}
