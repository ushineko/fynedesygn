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
	// caller's --yes: the window then shows only the result.
	SkipQuestion bool
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

	o         ConfirmOptions
	stage     int
	jobCancel context.CancelFunc
	closing   bool
	closed    bool
	result    Result
	err       error

	bar     *widget.ProgressBarInfinite
	message *widget.Label
	action  *widget.Button
	cancel  *widget.Button
	buttons *fyne.Container
	root    fyne.CanvasObject

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
	c := &ConfirmWindow{App: a, o: o}
	question := widget.NewLabelWithStyle(o.Question, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	question.Wrapping = fyne.TextWrapWord
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
	c.buttons = container.NewHBox(
		widgets.FixedWidth(c.cancel, widest(LabelCancel)),
		widgets.FixedWidth(c.action, widest(o.Action, LabelClose)),
	)
	// The bar's region has a fixed height, so showing the bar moves nothing.
	barRegion := widgets.FixedHeight(container.NewStack(c.bar), c.bar.MinSize().Height)
	c.root = container.NewBorder(
		container.NewPadded(container.NewVBox(question, widget.NewSeparator())),
		container.NewVBox(widget.NewSeparator(), container.NewPadded(container.NewBorder(nil, nil, nil, c.buttons, c.message))),
		nil, nil,
		container.NewPadded(container.NewBorder(nil, barRegion, nil, nil, container.NewVScroll(document(o.Detail)))),
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

// Message is the text on the message line.
func (c *ConfirmWindow) Message() string { return c.message.Text }

// Result is how the window ended; Outcome is Cancelled until it has.
func (c *ConfirmWindow) Result() Result { return c.result }

/*
Act is the action button. While asking, it runs the job. On the result, it
is Close.
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
closes. While the job runs, it asks, cancels the job, and closes when the
job returns. On the result, closing is the same as Close.
*/
func (c *ConfirmWindow) Cancel() {
	switch c.stage {
	case stageAsking:
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
	job := func() {
		err := c.o.Job(ctx)
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
	// arrived returns nil, and its work is done.
	if errors.Is(err, context.Canceled) {
		c.finish(Cancelled, nil)
		return
	}
	c.stage = stageResult
	c.err = err
	if err != nil {
		c.message.Importance = widgets.ImportanceFor(fd.StatusBad)
		c.message.SetText(fmt.Sprintf("Failed: %v", err))
	} else {
		c.message.Importance = widgets.ImportanceFor(fd.StatusGood)
		c.message.SetText(c.o.Done)
	}
	c.action.SetText(LabelClose)
	c.action.Importance = widget.HighImportance
	c.action.Enable()
	c.cancel.Disable()
	if c.closing {
		c.Act()
	}
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
