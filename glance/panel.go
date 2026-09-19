package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// MinWidth is the default floor for a glance window's width. Without a floor,
// a window with one card showing one short value is a chip. 260 is the
// monitor's, chosen so two device cells stay wide enough not to cut their
// names.
const MinWidth float32 = 260

// Panel is the stack of cards that makes up a glance window's content, and the
// thing that keeps the window the size of what it is drawing.
//
// There is no scroller. Content that does not fit is content that should have
// been hidden: a scrollbar is an instruction to interact, and nobody scrolls a
// widget they are reading past.
type Panel struct {
	cards []*Card
	stack *fyne.Container
	bg    *canvas.Rectangle
	root  *fyne.Container

	win      fyne.Window
	minWidth float32
}

// NewPanel builds an empty panel. minWidth of 0 means MinWidth.
func NewPanel(minWidth float32) *Panel {
	if minWidth <= 0 {
		minWidth = MinWidth
	}
	p := &Panel{
		stack:    container.NewVBox(),
		bg:       canvas.NewRectangle(theme.Color(theme.ColorNameBackground)),
		minWidth: minWidth,
	}
	// The background is an explicit rectangle rather than the window's own
	// fill because a glance window is designed opaque and its separation from
	// the desktop is contrast, not alpha (quirk 32). A nil fill colour is
	// invisible under the GL painter and a nil dereference under the software
	// one (quirk 25), so it is always set.
	p.root = container.NewStack(p.bg, container.NewPadded(p.stack))
	return p
}

// Add appends cards in the order they will be stacked, top to bottom. Each one
// is wired to resize the window when it appears or disappears.
func (p *Panel) Add(cards ...*Card) {
	for _, c := range cards {
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
	w.SetContent(p.root)
	p.Resize()
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
	p.win.Resize(p.Size())
}

// Restyle repaints the panel and every card in the current theme, after a
// scheme or text size change, and resizes: a larger face is a larger window.
func (p *Panel) Restyle() {
	p.bg.FillColor = theme.Color(theme.ColorNameBackground)
	p.bg.Refresh()
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
