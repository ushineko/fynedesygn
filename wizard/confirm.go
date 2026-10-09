package wizard

import (
	"context"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/dialogs"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

/*
ConfirmOptions describe a window that asks once, runs a job and reports
(spec 062): the shape of an uninstaller, where the whole decision is one
question and a wizard's page list would be noise.
*/
type ConfirmOptions struct {
	// AppID names the app, and on Wayland the window's app_id. Used by
	// NewConfirm.
	AppID string
	// Name is the window title.
	Name string
	// Icon is the window icon; nil leaves Fyne's.
	Icon fyne.Resource
	// Size is the window size; zero means ConfirmWidth x ConfirmHeight.
	Size fyne.Size
	// Appearance is applied by NewConfirm; nil means the platform default.
	Appearance *fdtheme.Appearance
	// Question is the bold first line, such as "Uninstall Hello 0.1.0?".
	Question string
	// Detail is Markdown under it: what will happen and what is left alone.
	Detail string
	// Action labels the button that runs the job, such as "Uninstall".
	Action string
	// Destructive styles the action button as destructive.
	Destructive bool
	// Job is the work. It returns an error wrapping context.Canceled when
	// it stopped because its context was cancelled.
	Job func(ctx context.Context) error
	// Done is the message line's text when the job succeeds.
	Done string
	// SkipQuestion runs the job as soon as the window is shown, for a
	// caller's --yes: the window then shows only the result. It does not
	// skip Then, which is a separate decision.
	SkipQuestion bool
	// Then is an optional second question, asked in the same window after
	// the job succeeds and built from what it found (spec 066).
	Then *ConfirmStep
}

// ConfirmStep is a confirm window's second question, such as "Remove the
// files the program left too?" after an uninstall. By the time it is asked
// the first job's work is done, so leaving without it is Finished.
type ConfirmStep struct {
	// Action labels the button that runs Job; Destructive styles it.
	Action      string
	Destructive bool
	// Decline labels the button that leaves without the step, such as
	// "Keep them". The default is Close.
	Decline string
	// Ask is called on the UI thread after the first job succeeds, and
	// returns the bold question and its Markdown detail. ok false skips the
	// step. It must not block: the job works out what the question needs.
	Ask func() (question, detail string, ok bool)
	// Job is the step's work, with the same rules as ConfirmOptions.Job.
	Job func(ctx context.Context) error
	// Done is the message line's text when Job succeeds.
	Done string
}

// The confirm window's default size.
const (
	ConfirmWidth  float32 = 560
	ConfirmHeight float32 = 280
)

// The stages of a confirm window, in the order it goes through them.
const (
	stageAsking = iota
	stageRunning
	stageResult
)

// ConfirmWindow is a running confirm window.
type ConfirmWindow struct {
	// App is the app the window runs in.
	App fyne.App
	// Window is nil for a HeadlessConfirm.
	Window fyne.Window

	o     ConfirmOptions
	stage int
	// inThen is true from the moment the step is asked.
	inThen    bool
	jobCancel context.CancelFunc
	closing   bool
	closed    bool
	result    Result
	err       error

	question *widget.Label
	detail   *container.Scroll
	bar      *widget.ProgressBarInfinite
	message  *widget.Label
	action   *widget.Button
	cancel   *widget.Button
	buttons  *fyne.Container
	root     fyne.CanvasObject

	confirm func(title, detail, ok string, do func())
	do      func(fn func())
}

// RunConfirm makes the window, shows it and blocks until it closes.
func RunConfirm(o ConfirmOptions) Result {
	c := NewConfirm(o)
	c.Window.ShowAndRun()
	return c.Result()
}

// NewConfirm creates the app, applies the appearance, and builds the
// window without showing it.
func NewConfirm(o ConfirmOptions) *ConfirmWindow {
	c := NewConfirmIn(newApp(o.AppID, o.Appearance, o.Icon), o)
	c.Window.SetMaster()
	return c
}

// NewConfirmIn builds the window over an app the caller already has, and
// leaves the app's theme alone. The window is not shown.
func NewConfirmIn(a fyne.App, o ConfirmOptions) *ConfirmWindow {
	c := newConfirm(a, o)
	c.Window = a.NewWindow(o.Name)
	if o.Icon != nil {
		c.Window.SetIcon(o.Icon)
	}
	c.do = fyne.Do
	c.confirm = func(title, detail, ok string, do func()) {
		dialogs.ConfirmDestructive(c.Window, title, detail, ok, do)
	}
	size := o.Size
	if size.IsZero() {
		size = fyne.NewSize(ConfirmWidth, ConfirmHeight)
	}
	c.Window.Resize(size)
	c.Window.SetFixedSize(true)
	c.Window.SetContent(c.root)
	c.Window.SetCloseIntercept(c.Cancel)
	return c.start()
}

// HeadlessConfirm builds the window's content with no window over an
// existing app (a test app). The job runs inline and the confirmation to
// stop it is accepted at once.
func HeadlessConfirm(a fyne.App, o ConfirmOptions) *ConfirmWindow {
	c := newConfirm(a, o)
	c.do = func(fn func()) { fn() }
	c.confirm = func(_, _, _ string, do func()) { do() }
	return c.start()
}

func newConfirm(a fyne.App, o ConfirmOptions) *ConfirmWindow {
	if o.Job == nil {
		panic("wizard: ConfirmOptions.Job is nil")
	}
	if o.Then != nil && (o.Then.Ask == nil || o.Then.Job == nil) {
		panic("wizard: ConfirmStep needs Ask and Job")
	}
	c := &ConfirmWindow{App: a, o: o}
	c.question = widget.NewLabelWithStyle(o.Question, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	c.question.Wrapping = fyne.TextWrapWord
	c.bar = widget.NewProgressBarInfinite()
	c.bar.Stop()
	c.bar.Hide()
	c.message = widget.NewLabel("")
	c.message.Truncation = fyne.TextTruncateEllipsis
	c.action = widget.NewButton(o.Action, c.Act)
	c.action.Importance = widget.HighImportance
	if o.Destructive {
		c.action.Importance = widget.DangerImportance
	}
	c.cancel = widget.NewButton(LabelCancel, c.Cancel)
	// The widths cover every label the buttons take, the step's too, so
	// nothing moves between stages (spec 062 R5, spec 066 R5).
	cancels, actions := []string{LabelCancel}, []string{o.Action, LabelClose}
	if o.Then != nil {
		cancels = append(cancels, o.Then.decline())
		actions = append(actions, o.Then.Action)
	}
	c.buttons = container.NewHBox(
		widgets.FixedWidth(c.cancel, widest(cancels...)),
		widgets.FixedWidth(c.action, widest(actions...)),
	)
	// The bar's region has a fixed height, so showing the bar moves nothing.
	barRegion := widgets.FixedHeight(container.NewStack(c.bar), c.bar.MinSize().Height)
	c.detail = container.NewVScroll(document(o.Detail))
	c.root = container.NewBorder(
		container.NewPadded(container.NewVBox(c.question, widget.NewSeparator())),
		container.NewVBox(widget.NewSeparator(), container.NewPadded(container.NewBorder(nil, nil, nil, c.buttons, c.message))),
		nil, nil,
		container.NewPadded(container.NewBorder(nil, barRegion, nil, nil, c.detail)),
	)
	return c
}

func (c *ConfirmWindow) start() *ConfirmWindow {
	if c.o.SkipQuestion {
		c.Act()
	}
	return c
}

// Content is the window's content; a HeadlessConfirm's test drives it.
func (c *ConfirmWindow) Content() fyne.CanvasObject { return c.root }

// Question is the bold question being asked.
func (c *ConfirmWindow) Question() string { return c.question.Text }

// Message is the text on the message line.
func (c *ConfirmWindow) Message() string { return c.message.Text }

// Result is how the window ended; Outcome is Cancelled until it has.
func (c *ConfirmWindow) Result() Result { return c.result }

/*
Act is the action button. While asking, it runs the job, or the step's job
when the step is asked. On the result, it is Close.
*/
func (c *ConfirmWindow) Act() {
	switch c.stage {
	case stageAsking:
		c.run()
	case stageResult:
		if c.err != nil {
			c.finish(Failed, c.err)
		} else {
			c.finish(Finished, nil)
		}
	}
}

/*
Cancel is the Cancel button and the title bar's close. While asking, it
closes: as Cancelled before the first job, as Finished when the step is
asked, since the first job's work is done. While a job runs, it asks,
cancels the job, and closes when the job returns. On the result, closing is
the same as Close.
*/
func (c *ConfirmWindow) Cancel() {
	switch c.stage {
	case stageAsking:
		if c.inThen {
			c.finish(Finished, nil)
			return
		}
		c.finish(Cancelled, nil)
	case stageRunning:
		c.confirm("Stop?", "The work in progress stops and is undone where it can be. The window closes when it has stopped.",
			"Stop", func() {
				c.closing = true
				if c.jobCancel != nil {
					c.jobCancel()
				}
			})
	case stageResult:
		c.Act()
	}
}

func (c *ConfirmWindow) run() {
	ctx, cancel := context.WithCancel(context.Background())
	c.jobCancel = cancel
	c.stage = stageRunning
	c.action.Disable()
	c.bar.Show()
	c.bar.Start()
	run := c.o.Job
	if c.inThen {
		run = c.o.Then.Job
	}
	job := func() {
		err := run(ctx)
		c.do(func() { c.jobDone(err) })
		cancel()
	}
	if c.Window == nil {
		job()
		return
	}
	go job()
}

func (c *ConfirmWindow) jobDone(err error) {
	c.bar.Stop()
	c.bar.Hide()
	// The job's own error decides (R4): one that finished as Cancel
	// arrived returns nil, and its work is done. A cancelled step leaves
	// the first job's work standing, so it ends as Finished (spec 066 R4).
	if errors.Is(err, context.Canceled) {
		if c.inThen {
			c.finish(Finished, nil)
		} else {
			c.finish(Cancelled, nil)
		}
		return
	}
	c.err = err
	if err != nil {
		c.message.Importance = widgets.ImportanceFor(fd.StatusBad)
		c.message.SetText(fmt.Sprintf("Failed: %v", err))
	} else {
		c.message.Importance = widgets.ImportanceFor(fd.StatusGood)
		if c.inThen {
			c.message.SetText(c.o.Then.Done)
		} else {
			c.message.SetText(c.o.Done)
		}
	}
	if err == nil && !c.inThen && !c.closing && c.askThen() {
		return
	}
	c.stage = stageResult
	c.action.SetText(LabelClose)
	c.action.Importance = widget.HighImportance
	c.action.Enable()
	c.cancel.Disable()
	if c.closing {
		c.Act()
	}
}

// askThen asks the step, if there is one and its Ask says so, and reports
// whether it did. The message line keeps the first job's Done.
func (c *ConfirmWindow) askThen() bool {
	if c.o.Then == nil {
		return false
	}
	q, d, ok := c.o.Then.Ask()
	if !ok {
		return false
	}
	c.inThen = true
	c.stage = stageAsking
	c.question.SetText(q)
	c.detail.Content = document(d)
	c.detail.Refresh()
	c.detail.ScrollToTop()
	c.action.SetText(c.o.Then.Action)
	c.action.Importance = widget.HighImportance
	if c.o.Then.Destructive {
		c.action.Importance = widget.DangerImportance
	}
	c.action.Enable()
	c.cancel.SetText(c.o.Then.decline())
	c.cancel.Enable()
	return true
}

func (s *ConfirmStep) decline() string {
	if s.Decline != "" {
		return s.Decline
	}
	return LabelClose
}

func (c *ConfirmWindow) finish(o Outcome, err error) {
	if c.closed {
		return
	}
	c.closed = true
	c.result = Result{Outcome: o, Err: err, Checks: map[string]bool{}}
	if c.Window != nil {
		c.Window.Close()
	}
}
