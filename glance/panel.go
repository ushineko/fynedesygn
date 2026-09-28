package glance

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// MinWidth is the default floor for a glance window's width. Without a floor,
// a window with one card showing one short value is a chip. 260 is the
// monitor's, chosen so two device cells stay wide enough not to cut their
// names.
const MinWidth float32 = 260

/*
NoMinWidth asks for no floor at all: the panel is exactly as wide as the widest
thing in it.

The default floor is a guard against a panel of one short reading looking like
a chip, and it costs nothing while the content is wider than it. A panel whose
cards have been made narrow pays for it in dead space -- a consumer measured
its widest card at 218 and got a 260-pixel window, forty-two pixels of panel
with nothing in it down the right-hand side.

A consumer that has looked at what its own cards measure, and would rather the
window fitted them, asks for this.
*/
const NoMinWidth float32 = -1

// Panel is the stack of cards that makes up a glance window's content, and the
// thing that keeps the window the size of what it is drawing.
//
// There is no scroller. Content that does not fit is content that should have
// been hidden: a scrollbar is an instruction to interact, and nobody scrolls a
// widget they are reading past.
type Panel struct {
	themed

	cards []*Card
	stack *fyne.Container
	bg    *canvas.Rectangle
	root  *fyne.Container

	win      fyne.Window
	minWidth float32

	// translucent draws the panel's own background clear, for a window that
	// got a transparent framebuffer.
	translucent bool

	// override is the panel's own theme, when it has been given one. Nil
	// until SetTheme is called, so a panel that never asks carries no extra
	// widget and behaves exactly as it always did.
	override *container.ThemeOverride

	// resizable is set from the window's Options. A resizable panel never
	// pulls the window narrower than the user has made it; see Resize.
	resizable bool

	// asked is the width this panel last requested, and sized says whether it
	// has ever requested one. Together they are how Resize tells a width the
	// user chose from a width nobody chose; see Resize.
	asked float32
	sized bool
}

// NewPanel builds an empty panel. minWidth of 0 means MinWidth.
func NewPanel(minWidth float32) *Panel {
	switch {
	case minWidth < 0:
		// NoMinWidth: the content decides, and Size clamps nothing.
		minWidth = 0
	case minWidth == 0:
		minWidth = MinWidth
	}
	p := &Panel{
		stack:    container.New(layout.NewCustomPaddedVBoxLayout(CardGap())),
		bg:       canvas.NewRectangle(nil),
		minWidth: minWidth,
	}
	// The background is an explicit rectangle rather than the window's own
	// fill because a glance window is designed opaque and its separation from
	// the desktop is contrast, not alpha (quirk 32). A nil fill colour is
	// invisible under the GL painter and a nil dereference under the software
	// one (quirk 25), so it is always set.
	// The stack is not padded away from the window's edge. In the monitor a
	// card runs to the edge and the window *is* the cards; a ring of window
	// background around them reads as a frame the panel does not have. The
	// space between one card and the next is CardGap, which is the only place
	// the background shows.
	p.bg.FillColor = p.BackgroundColour()
	p.root = container.NewStack(p.bg, p.stack)
	return p
}

// Add appends cards in the order they will be stacked, top to bottom. Each one
// is wired to resize the window when it appears or disappears.
func (p *Panel) Add(cards ...*Card) {
	for _, c := range cards {
		c.SetTheme(p.th)
		p.cards = append(p.cards, c)
		p.stack.Add(c.Object())

		prev := c.OnDrawnChanged
		c.OnDrawnChanged = func(drawn bool) {
			if prev != nil {
				prev(drawn)
			}
			p.Resize()
		}
	}

	// A card added after the override was built keeps the default theme until
	// the override is refreshed. Fyne's own documentation for it says so, and
	// every consumer adds its cards after the window exists, so this is the
	// ordinary path rather than an edge of it.
	if p.override != nil {
		p.override.Refresh()
	}
}

// Cards are the panel's cards in order, for tests and for a context menu
// building a toggle per card.
func (p *Panel) Cards() []*Card { return p.cards }

// Drawn counts the cards currently on screen.
func (p *Panel) Drawn() int {
	n := 0
	for _, c := range p.cards {
		if c.Drawn() {
			n++
		}
	}
	return n
}

// Attach binds the panel to the window it will size. A panel with no window
// still works and still measures; it just has nothing to resize, which is what
// a headless test has.
func (p *Panel) Attach(w fyne.Window) {
	p.win = w
	w.SetContent(p.content())
	p.Resize()
}

// content is what the window is given: the panel, inside its own theme when it
// has one.
func (p *Panel) content() fyne.CanvasObject {
	if p.override != nil {
		return p.override
	}
	return p.root
}

/*
SetTheme gives the panel a face of its own: its own scheme, its own family,
its own size, independent of the application's.

**This is how a program gives its panel and its settings window different
faces, and it is the panel that takes the separate one.** The application's
theme has to belong to the window that has *overlays* -- a dialog, a
widget.Select's dropdown, a context menu -- because those are added to the
canvas's overlay stack rather than to a window's content, so nothing can
override them. A panel has no overlays. So the panel carries its own theme and
the window keeps the application's, and everything in that window is right,
dialogs included.

This used to say the opposite, and the opposite did not work. It reached
standard widgets through a subtree override and left the cards reading the
application's theme, so the padding changed and the text did not. Both halves
happen here now: every card is handed the theme, and the override stays for
anything inside a panel that is a standard widget.

nil takes the theme away again and the panel follows the application, which is
what it does until this is called and what every panel that has not asked for
anything else wants.

**A panel with a theme of its own must be refreshed when cards are added**, and
Add does that.
*/
func (p *Panel) SetTheme(th fyne.Theme) {
	p.themed.SetTheme(th)

	switch {
	case th == nil:
		p.override = nil
	case p.override == nil:
		p.override = container.NewThemeOverride(p.root, th)
	default:
		p.override.Theme = th
		p.override.Refresh()
	}

	// The cards are the half the override cannot do: they are canvas objects
	// and read the theme they are given rather than the application's.
	for _, c := range p.cards {
		c.SetTheme(th)
	}

	if p.win != nil {
		p.win.SetContent(p.content())
	}
	p.Restyle()
}

// Size is the size the window should be: the content's minimum, widened to the
// panel's floor. It is exported so a test can assert the window shrank without
// needing a compositor.
func (p *Panel) Size() fyne.Size {
	s := p.root.MinSize()
	if s.Width < p.minWidth {
		s.Width = p.minWidth
	}
	return s
}

// Resize takes the window to the size of what it is drawing.
//
// This is not decoration. A fixed-size Fyne window grows to fit its content on
// its own and never shrinks back: fitContent only ever raises the requested
// size, and Resize is the only thing that lowers it (quirk 34). Without this
// call a card that hides leaves the window at its high-water mark, with an
// empty band where the card was.
//
// It runs now rather than on the next frame. A window that resizes a frame
// after the cause resizes visibly.
func (p *Panel) Resize() {
	if p.win == nil {
		return
	}
	p.stack.Refresh()

	want := p.Size()

	/*
		The user's width is theirs, and the trick is telling it from a width
		nobody chose.

		Comparing against the window's *current* width alone cannot: before
		the first Resize the window is whatever Fyne made it, which is not a
		choice and is usually wider than the content. Honouring it locked that
		first guess in for the life of the program -- a consumer whose widest
		card measured 218 opened at 298 and stayed there, and the only way to
		get the window it wanted was to drag it narrower by hand every time.

		So the first Resize always takes the content, and afterwards a width
		that is not the one this panel last asked for is a width something
		else set: the user, or the compositor on their behalf.

		The height follows the content either way, which is what quirk 34 is
		about: a card that hides leaves a band of empty window behind it
		unless something lowers the requested size, and nobody chose that
		band.
	*/
	if p.resizable && p.sized {
		if current := p.win.Canvas().Size().Width; current > want.Width && current != p.asked {
			want.Width = current
		}
	}
	p.asked, p.sized = want.Width, true

	p.win.Resize(want)
}

// Restyle repaints the panel and every card in the current theme, after a
// scheme or text size change, and resizes: a larger face is a larger window.
func (p *Panel) Restyle() {
	p.bg.FillColor = p.BackgroundColour()
	p.bg.Refresh()
	// The gap is a factor of the text size; see Card's margins for why the
	// layout is replaced rather than refreshed.
	p.stack.Layout = layout.NewCustomPaddedVBoxLayout(p.gap())
	for _, c := range p.cards {
		c.Restyle()
	}
	p.Resize()
}

// Overlay stacks an object over the whole panel, for something that has to
// see the panel's full area rather than sit in the card stack: the context
// menu's tap catcher is the one case. It is drawn last, which is what makes
// it win the pointer (quirk 24).
func (p *Panel) Overlay(o fyne.CanvasObject) { p.root.Add(o) }

// Content is the panel's root object, for a caller building its own window.
func (p *Panel) Content() fyne.CanvasObject { return p.root }

/*
BackgroundColour is what the panel paints behind its cards.

Clear when the window got a transparent framebuffer, the scheme's colour
otherwise.

**This is deliberately the panel's own background and not the app's theme.** A
Fyne theme is app-wide, so a program with a second window — a preferences
window, an indicator's settings — had that window's background turned
transparent too, and a window that is not translucent renders a transparent
background as black. It also meant anything that legitimately set a theme
afterwards dropped the transparency with it, and the glance window quietly
stopped being see-through.
*/
func (p *Panel) BackgroundColour() color.Color {
	if p.translucent {
		return color.Transparent
	}
	return p.colour(theme.ColorNameBackground)
}

// SetTranslucent draws the panel's background clear. Window calls it when the
// framebuffer grant comes; there is nothing for a caller to do.
func (p *Panel) SetTranslucent(on bool) {
	p.translucent = on
	p.bg.FillColor = p.BackgroundColour()
	p.bg.Refresh()
}
