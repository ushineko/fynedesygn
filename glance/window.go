package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/widgets"
)

// Options describe a glance window. Every field has a working zero value
// except Title, which names the window for the taskbar and for a compositor
// rule that matches on it.
type Options struct {
	// Title is the window title. On Wayland the compositor matches the window
	// by the app ID rather than this, and Fyne takes the app ID from
	// fyne.App.UniqueID (see docs/glance.md, Desktop integration).
	Title string

	// MinWidth is the floor for the window's width. Zero means MinWidth.
	MinWidth float32

	// Decorated leaves the window its titlebar and border. The default is a
	// frameless window, which is what a glance window is; this is the escape
	// hatch for a desktop where a frameless window cannot be moved at all.
	Decorated bool

	// OnTop asks the window manager to keep the window above others. The
	// request is made before the window is shown, which is the only time it is
	// honoured, and the window manager may still decide otherwise.
	OnTop bool

	// Menu builds the context menu, called fresh on every secondary tap so it
	// can tick the current value of what it shows. Nil means no menu, which
	// for a glance window means no interface at all: everything a glance
	// window configures is configured here.
	Menu func() *fyne.Menu
}

// Window is a glance window: a frameless, fixed-size, always-on-top panel
// holding a stack of cards.
//
// It is a concrete type for the same reason shell.Shell is: the cards are the
// program's own code, and an interface here would buy nothing.
type Window struct {
	win   fyne.Window
	panel *Panel
	menu  func() *fyne.Menu
}

// NewWindow builds the window and its panel. The window is not shown; add
// cards to Panel first, so the first thing drawn is the real size rather than
// an empty frame that grows.
//
// Frameless needs the desktop driver. On any other driver — the test driver
// above all — the window is an ordinary one and everything else behaves the
// same, so a headless test exercises the real panel.
func NewWindow(a fyne.App, o Options) *Window {
	w := &Window{menu: o.Menu}

	drv, isDesktop := a.Driver().(desktop.Driver)
	switch {
	case o.Decorated || !isDesktop:
		w.win = a.NewWindow(o.Title)
	default:
		// CreateSplashWindow is the only way to an undecorated window in
		// Fyne 2.8: there is no per-window decoration setting. It also
		// unpads and centres, which is what a glance window wants on a
		// first run anyway.
		w.win = drv.CreateSplashWindow()
		w.win.SetTitle(o.Title)
	}

	// Fixed size, because nothing here is resizable by hand: the window is
	// whatever its content measures. It still has to be resized explicitly
	// when a card goes away (quirk 34), which Panel does.
	w.win.SetFixedSize(true)
	w.win.SetMaster()

	if o.OnTop {
		// Before Show, which is the contract: RequestAlwaysOnTop sets a
		// window hint that is read when the window is created.
		if dw, ok := w.win.(desktop.Window); ok {
			dw.RequestAlwaysOnTop()
		}
	}

	w.panel = NewPanel(o.MinWidth)
	w.panel.Attach(w.win)

	if o.Menu != nil {
		w.panel.stack.Add(newMenuCatcher(w))
	}
	return w
}

// Panel is the card stack. Add cards to it before showing the window.
func (w *Window) Panel() *Panel { return w.panel }

// Window is the Fyne window, for the few things this package does not wrap:
// the icon, the close intercept, a keyboard shortcut.
func (w *Window) Window() fyne.Window { return w.win }

// ShowAndRun shows the window and runs the event loop. It returns when the
// window closes.
func (w *Window) ShowAndRun() {
	w.panel.Resize()
	w.win.ShowAndRun()
}

// ShowMenu opens the context menu at a position on the window's canvas. It is
// what the catcher calls and is exported so a program can bind the menu to a
// key as well as to a tap.
func (w *Window) ShowMenu(pos fyne.Position) {
	if w.menu == nil {
		return
	}
	// Anything that opens over the window takes its tips down first: clicking
	// leaves the pointer where it was, so nothing else tells a tip that the
	// control under it has been used (quirk 27).
	widgets.HideTips()
	widget.ShowPopUpMenuAtPosition(w.menu(), w.win.Canvas(), pos)
}

// menuCatcher is a zero-height object covering the panel that turns a
// secondary tap anywhere in the window into the context menu.
//
// It is a widget rather than a handler on the panel because Fyne routes
// pointer events to objects, and there is no window-level tap callback.
type menuCatcher struct {
	widget.BaseWidget
	owner *Window
}

func newMenuCatcher(owner *Window) *menuCatcher {
	c := &menuCatcher{owner: owner}
	c.ExtendBaseWidget(c)
	return c
}

// TappedSecondary opens the menu. The catcher is deliberately not Tappable, so
// an ordinary click walks past it to whatever is underneath (quirk 24).
func (c *menuCatcher) TappedSecondary(e *fyne.PointEvent) {
	c.owner.ShowMenu(e.AbsolutePosition)
}

func (c *menuCatcher) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widgets.FixedHeight(widgets.Dim(""), 0))
}

var _ fyne.SecondaryTappable = (*menuCatcher)(nil)
