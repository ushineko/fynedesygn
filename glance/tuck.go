package glance

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

/*
A panel tucks into a square at its corner (spec 081).

A glance panel stays on top, and while it covers something a person is
working on there was no way to put it aside short of quitting. Tucked, the
window is a small square bearing the program's icon at the cards' bottom
corner; a click on the square brings the panel back. The cards and drawers
are hidden, not closed, and come back as they were.

**The square is at a corner of the window, and the corner is held by
whatever moves the window**: the window itself on Windows, the compositor on
Plasma (kwin.HoldCornerScript). The cards' bottom corner on the screen-edge
side is always a corner of the window: with drawers open it is the one away
from them, and with none the window is the cards.

The square moves as the panel does, by Meta-drag or a drag on Windows, and
the panel unfolds from wherever it was put.
*/

// TuckSizeInEms is the side of the tucked square in text sizes.
const TuckSizeInEms = 3

// tucking is a panel's tuck state and the objects that draw it.
type tucking struct {
	tucked bool
	corner Side

	// held says something keeps the window's corner while it is resized:
	// the compositor's script (SetCornerHeld) or the window itself.
	held bool

	// square is the tucked window's content, face its ground and icon the
	// program's icon on it.
	square *fyne.Container
	face   *canvas.Rectangle
	icon   *canvas.Image
	res    fyne.Resource

	// glyph is the hover control that asks to tuck, drawn over the cards'
	// top-right corner while the pointer is over them; layer holds it.
	tuckable bool
	glyph    *fyne.Container
	glyphBg  *canvas.Rectangle
	glyphBar *canvas.Rectangle
	layer    *fyne.Container

	// untucking is set while the window unfolds, so the square's width is
	// not read as a width somebody chose (resize).
	untucking bool

	onTuck  func(bool)
	onAsked func()
}

// buildTuck makes the square and the glyph, hidden.
func (p *Panel) buildTuck() {
	t := &p.tuck
	t.face = canvas.NewRectangle(nil)
	t.icon = canvas.NewImageFromResource(nil)
	t.icon.FillMode = canvas.ImageFillContain
	t.square = container.New(&squareLayout{panel: p}, t.face, t.icon)
	t.square.Hide()

	t.glyphBg = canvas.NewRectangle(nil)
	t.glyphBar = canvas.NewRectangle(nil)
	t.glyph = container.NewWithoutLayout(t.glyphBg, t.glyphBar)
	t.glyph.Hide()
	t.layer = container.New(&glyphLayout{panel: p}, t.glyph)
}

/*
SetTuckIcon is the icon the tucked square bears. Without one it is the
application's (fyne.App.Icon).
*/
func (p *Panel) SetTuckIcon(res fyne.Resource) {
	p.tuck.res = res
	p.restyleTuck()
}

/*
SetCornerHeld says the compositor keeps a bottom corner of the window where
it is while the window is resized: on Plasma, kwin.HoldCornerScript is
loaded. A window that moves itself (Windows) needs nobody to say so.

**A panel whose corner is not held does not tuck**: the square would land
wherever the compositor put the shrunken window, which on Plasma 6.8 is its
middle.
*/
func (p *Panel) SetCornerHeld(held bool) { p.tuck.held = held }

// CanTuck reports whether the window's corner can be held while it is
// resized, so a program knows whether to offer tucking.
func (p *Panel) CanTuck() bool { return p.selfMoving || p.tuck.held || p.win == nil }

/*
TuckCorner is the corner the panel would tuck into, given the room between
the window and each edge of its screen's work area (as Fits takes it): with
drawers open the corner away from them, and with none the corner on the side
with less room, which is the screen's edge the panel stands against.

A program on Plasma asks it before Tuck, to load the corner script for that
corner.
*/
func (p *Panel) TuckCorner(left, right float32) Side {
	if len(p.row) > 0 {
		if p.side == Left {
			return Right
		}
		return Left
	}
	if right <= left {
		return Right
	}
	return Left
}

/*
Tuck shows the square in place of the cards and drawers, at the corner
given (TuckCorner), and resizes the window to it. It reports false and
changes nothing when the corner cannot be held (CanTuck).

While tucked a reading's Resize keeps the square, and drawers and cards may
change without resizing the window.
*/
func (p *Panel) Tuck(corner Side) bool {
	t := &p.tuck
	if t.tucked {
		return true
	}
	if !p.CanTuck() {
		return false
	}
	before := p.windowSize()
	t.tucked, t.corner = true, corner
	t.glyph.Hide()
	p.body.Hide()
	p.restyleTuck()
	t.square.Show()
	after := p.squareSize()
	if p.win != nil {
		// Shrinking: resize, then move the corner back where it was.
		p.win.Resize(after)
		if p.selfMoving {
			moveCorner(p.win, corner, before, after)
		}
	}
	p.root.Refresh()
	if t.onTuck != nil {
		t.onTuck(true)
	}
	return true
}

/*
TuckHere is Tuck at the corner the room the window reads itself gives
(Windows). Where it cannot read the room, known is false: ask the compositor,
then TuckCorner and Tuck.
*/
func (p *Panel) TuckHere() (ok, known bool) {
	if p.win == nil {
		return false, false
	}
	left, right, have := roomBeside(p.win)
	if !have {
		return false, false
	}
	return p.Tuck(p.TuckCorner(left, right)), true
}

/*
Untuck puts the cards and drawers back and the window to their size, holding
the square's corner, so a square nobody moved brings the cards back where
they were. A click on the square calls it.
*/
func (p *Panel) Untuck() {
	t := &p.tuck
	if !t.tucked {
		return
	}
	before := p.squareSize()
	t.tucked = false
	t.square.Hide()
	p.body.Show()
	after := p.Size()
	if p.resizable && p.chosen > 0 {
		after.Width = max(after.Width, p.chosen+p.drawersExtra())
	}
	if p.win != nil && p.selfMoving {
		// Growing: move first, so the cards are never drawn off the window.
		moveCorner(p.win, t.corner, before, after)
	}
	t.untucking = true
	p.Resize()
	t.untucking = false
	p.root.Refresh()
	if t.onTuck != nil {
		t.onTuck(false)
	}
}

// Tucked reports whether the panel is tucked.
func (p *Panel) Tucked() bool { return p.tuck.tucked }

// SetOnTuck is called after the panel tucks (true) and untucks (false), on
// the UI thread, so a program stops and starts its readings.
func (p *Panel) SetOnTuck(f func(tucked bool)) { p.tuck.onTuck = f }

/*
SetTuckable shows the tuck glyph over the cards' top-right corner while the
pointer is over the cards, or never. A click on it calls asked, which is the
program's: on Plasma it asks the compositor for the room and loads the
corner script, then calls TuckCorner and Tuck; on Windows it calls TuckHere.
*/
func (p *Panel) SetTuckable(on bool, asked func()) {
	p.tuck.tuckable, p.tuck.onAsked = on, asked
	if !on {
		p.tuck.glyph.Hide()
	}
}

// Tuckable reports whether the tuck glyph is offered (SetTuckable).
func (p *Panel) Tuckable() bool { return p.tuck.tuckable }

// hoverTuck shows the glyph while the pointer is over the cards and hides
// it otherwise.
func (p *Panel) hoverTuck(at fyne.Position, in bool) {
	t := &p.tuck
	show := in && t.tuckable && !t.tucked && p.over(p.stack, at)
	if show == t.glyph.Visible() {
		return
	}
	if show {
		p.restyleTuck()
		t.glyph.Show()
	} else {
		t.glyph.Hide()
	}
	t.layer.Refresh()
}

// tuckTapped handles a click while tucked or on the glyph, and reports
// whether it did.
func (p *Panel) tuckTapped(at fyne.Position) bool {
	t := &p.tuck
	if t.tucked {
		p.Untuck()
		return true
	}
	if t.glyph.Visible() && p.over(t.glyph, at) {
		t.glyph.Hide()
		if t.onAsked != nil {
			t.onAsked()
		}
		return true
	}
	return false
}

// over reports whether a canvas position is on an object.
func (p *Panel) over(o fyne.CanvasObject, at fyne.Position) bool {
	if p.win == nil || !o.Visible() {
		return false
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(o)
	size := o.Size()
	return at.X >= pos.X && at.X < pos.X+size.Width && at.Y >= pos.Y && at.Y < pos.Y+size.Height
}

// squareSize is the tucked window's size.
func (p *Panel) squareSize() fyne.Size {
	s := p.textSize() * TuckSizeInEms
	return fyne.NewSize(s, s)
}

// windowSize is the window's size now, or the panel's without a window.
func (p *Panel) windowSize() fyne.Size {
	if p.win != nil {
		return p.win.Canvas().Size()
	}
	return p.Size()
}

// glyphSize is the side of the tuck glyph.
func (p *Panel) glyphSize() float32 { return p.textSize() * 1.25 }

// restyleTuck paints the square and the glyph in the current theme.
func (p *Panel) restyleTuck() {
	t := &p.tuck
	if t.face == nil {
		return
	}
	t.face.FillColor = p.colour(theme.ColorNameButton)
	t.face.StrokeColor = p.colour(theme.ColorNameSeparator)
	t.face.StrokeWidth = 1
	t.face.CornerRadius = CardRadius
	res := t.res
	if res == nil && fyne.CurrentApp() != nil {
		res = fyne.CurrentApp().Icon()
	}
	if t.icon.Resource != res {
		t.icon.Resource = res
		t.icon.Refresh()
	}
	t.face.Refresh()

	t.glyphBg.FillColor = p.colour(theme.ColorNameButton)
	t.glyphBg.StrokeColor = p.colour(theme.ColorNameSeparator)
	t.glyphBg.StrokeWidth = 1
	t.glyphBg.CornerRadius = p.glyphSize() / 4
	t.glyphBar.FillColor = p.colour(theme.ColorNameForeground)
	g := p.glyphSize()
	t.glyphBg.Resize(fyne.NewSize(g, g))
	bar := fyne.NewSize(g*0.5, max(1, float32(math.Round(float64(g*0.1)))))
	t.glyphBar.Resize(bar)
	t.glyphBar.Move(fyne.NewPos((g-bar.Width)/2, g*0.68))
	t.glyph.Resize(fyne.NewSize(g, g))
	t.glyphBg.Refresh()
	t.glyphBar.Refresh()
}

// squareLayout fills the window with the square's ground and puts the icon
// in its middle, inset by a third of a text size.
type squareLayout struct{ panel *Panel }

func (l *squareLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 0) }

func (l *squareLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
	inset := l.panel.textSize() * 0.4
	objects[1].Move(fyne.NewPos(inset, inset))
	objects[1].Resize(fyne.NewSize(max(0, size.Width-2*inset), max(0, size.Height-2*inset)))
}

// glyphLayout stands the glyph in the cards' top-right corner, inside the
// first card's margins, over the right end of its heading where the stale
// mark goes and the title does not.
type glyphLayout struct{ panel *Panel }

func (l *glyphLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 0) }

func (l *glyphLayout) Layout(objects []fyne.CanvasObject, _ fyne.Size) {
	p := l.panel
	g := p.glyphSize()
	at := p.stack.Position()
	x := at.X + p.stack.Size().Width - p.padH()*0.5 - g
	y := at.Y + p.padTop()*0.5
	objects[0].Move(fyne.NewPos(x, y))
	objects[0].Resize(fyne.NewSize(g, g))
}
