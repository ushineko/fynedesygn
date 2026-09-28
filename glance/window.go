package glance

import (
	"errors"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver"
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

	// MinWidth is the floor for the window's width. Zero means the package's
	// MinWidth; NoMinWidth means no floor, and the window is as wide as the
	// widest card in it.
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

	// Translucent asks for a window the desktop shows through: the panel's
	// background is drawn transparent and only the cards are painted, which
	// is the shape the archetype comes from.
	//
	// It is a request. The window is created with GLFW's transparent
	// framebuffer hint, and the hint is refused on some drivers, so the grant
	// is probed for first and an opaque window is what a refusal gives. Ask
	// Translucent() after ShowAndRun has started for the answer.
	//
	// The app's theme is wrapped by ShowAndRun when the grant comes, so the
	// caller sets its theme as usual and does not have to know about this.
	Translucent bool

	// Secondary marks a glance window that belongs to a program with a main
	// window of its own, such as an indicator. It is not the master window,
	// so closing or hiding it does not end the program.
	Secondary bool

	// Resizable lets the window manager resize the window.
	//
	// A glance window is its content by default, and that is the right
	// default: the window is whatever its readings measure and nothing has to
	// be dragged to fit. What it costs is that the compositor's own resize is
	// greyed out — a frameless window still has its window menu, and Resize in
	// it does nothing, because a fixed-size window tells the window manager it
	// will not take one.
	//
	// Set this and the window takes a size from the user and keeps it. It can
	// still never be *narrower* than its content: Fyne clamps to the minimum
	// size, which is the right floor and is not this package's to override.
	//
	// The shrink-back that quirk 34 needs still happens when a card hides —
	// otherwise the empty band it describes comes back — but it shrinks only
	// as far as the user's own width, never past it.
	Resizable bool
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

	app         fyne.App
	wanted      bool
	translucent bool
}

// NewWindow builds the window and its panel. The window is not shown; add
// cards to Panel first, so the first thing drawn is the real size rather than
// an empty frame that grows.
//
// Frameless needs the desktop driver. On any other driver — the test driver
// above all — the window is an ordinary one and everything else behaves the
// same, so a headless test exercises the real panel.
func NewWindow(a fyne.App, o Options) *Window {
	w := &Window{menu: o.Menu, app: a, wanted: o.Translucent}

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
	w.win.SetFixedSize(!o.Resizable)
	if !o.Secondary {
		w.win.SetMaster()
	}

	if o.OnTop {
		// Before Show, which is the contract: RequestAlwaysOnTop sets a
		// window hint that is read when the window is created.
		if dw, ok := w.win.(desktop.Window); ok {
			dw.RequestAlwaysOnTop()
		}
	}

	w.panel = NewPanel(o.MinWidth)
	w.panel.resizable = o.Resizable
	w.panel.Attach(w.win)

	if o.Menu != nil {
		// Over the whole panel, not beside the cards. A catcher in the card
		// stack would take up a card's worth of height and would only see
		// taps inside its own strip; stacked over, it sees the window.
		w.panel.Overlay(newMenuCatcher(w))
	}
	return w
}

// Panel is the card stack. Add cards to it before showing the window.
func (w *Window) Panel() *Panel { return w.panel }

// Window is the Fyne window, for the few things this package does not wrap:
// the icon, the close intercept, a keyboard shortcut.
func (w *Window) Window() fyne.Window { return w.win }

// ErrOpacityNeedsCompositor is returned by SetOpacity on a window the program
// cannot fade itself. Wayland has no protocol for a client's own opacity —
// GLFW's Wayland backend answers GLFW_FEATURE_UNAVAILABLE for the same
// request — so the compositor has to be asked. On Plasma that is
// kwin.OpacityScript, run over the session bus; this package returns the
// script as text rather than making the call, for the reason
// kwin.ReconfigureCall gives.
var ErrOpacityNeedsCompositor = errors.New(
	"this window cannot set its own opacity; ask the compositor (see glance/kwin.OpacityScript)")

// SetOpacity fades the whole window, after the toolkit has drawn it. It is a
// different thing from Options.Translucent, and the two compose: translucency
// decides which pixels are drawn at all, opacity decides how much of what was
// drawn survives.
//
// On X11 the program sets its own, live, by writing the property a compositing
// window manager reads. Everywhere else it returns
// ErrOpacityNeedsCompositor, and the caller asks the compositor.
//
// opacity is 0..1 and is clamped. There is no reading it back: the property is
// a request to the window manager and the window manager does not answer.
func (w *Window) SetOpacity(opacity float32) error {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}

	native, ok := w.win.(driver.NativeWindow)
	if !ok {
		return ErrOpacityNeedsCompositor
	}

	err := ErrOpacityNeedsCompositor
	native.RunNative(func(ctx any) {
		// By value, not by pointer. Fyne passes `f(context)` with a
		// driver.X11WindowContext value (internal/driver/glfw/
		// window_x11wayland.go); an assertion on the pointer type compiles,
		// never matches, and looks exactly like a Wayland session.
		x11, ok := ctx.(driver.X11WindowContext)
		if !ok {
			return
		}
		if setWindowOpacity(x11.WindowHandle, opacity) {
			err = nil
		}
	})
	return err
}

// Translucent reports whether the window actually got a transparent
// framebuffer. It is false until ShowAndRun has shown the window, and false
// afterwards on any desktop that refused the request.
func (w *Window) Translucent() bool { return w.translucent }

// ShowAndRun shows the window and runs the event loop. It returns when the
// window closes.
//
// A translucent window is shown differently, and has to be. The hint that
// makes a window see-through is set on GLFW, which is initialised by the event
// loop, so a window shown before the loop starts is created before there is
// anything to ask. Show is therefore queued from a goroutine — fyne.Do run
// from the main goroutine before the loop is running executes immediately,
// which would be the same problem — and the loop runs it as its first work.
func (w *Window) ShowAndRun() {
	w.panel.Resize()
	if !w.wanted {
		w.win.ShowAndRun()
		return
	}

	go fyne.Do(func() {
		if grantTranslucent() {
			w.translucent = true
			// Before the window exists: the first frame is already the real
			// one, and what it is cleared with is decided here.
			//
			// The panel's own background, not the app's theme. A theme is
			// app-wide, so a program with a second window had that window
			// turned transparent too -- and a window that is not translucent
			// draws a transparent background as black.
			w.panel.SetTranslucent(true)
			w.panel.Restyle()
		}
		w.win.Show()

		// The hint has been taken up, so it is put back. Fyne creates the
		// GLFW window inside Show -- this body is already on the main
		// goroutine, so Fyne's EnsureMain runs inline rather than queueing --
		// and a hint left set is inherited by every window created afterwards,
		// including the ones this library did not make. Clearing it any
		// earlier costs the panel its own translucency, silently and only on
		// the desktops that would have granted it.
		clearTranslucent()
	})
	w.app.Run()
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

// menuCatcher is a transparent object stacked over the panel that turns a
// secondary tap anywhere in the window into the context menu.
//
// It is a widget rather than a handler on the panel because Fyne routes
// pointer events to objects and there is no window-level tap callback, and it
// is stacked over the content rather than placed beside it because a pointer
// event goes to the last match in the tree walk (quirk 24). It is not
// Tappable, so an ordinary click walks past it to whatever is underneath.
//
// Its rectangle is filled with color.Transparent rather than left nil: a nil
// fill is invisible under the GL painter and a nil dereference under the
// software one, which would make the window impossible to render to an image
// (quirk 25).
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
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

// MinSize is zero: the catcher is stacked over the panel and must never be the
// reason the window is a pixel bigger than its cards.
func (c *menuCatcher) MinSize() fyne.Size { return fyne.Size{} }

var _ fyne.SecondaryTappable = (*menuCatcher)(nil)
