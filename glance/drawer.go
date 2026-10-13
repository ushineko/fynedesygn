package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// Side is the side of the card stack a drawer opens on.
type Side int

const (
	// Right opens the drawer to the right of the cards. The window's origin
	// stays where it is and the window grows.
	Right Side = iota
	// Left opens it to the left: the window grows leftwards, so its right
	// edge stays where it is and the cards do not move on the screen.
	Left
)

// DrawerWidthInEms is the drawer's width in text sizes: a name, a figure and
// something between them, a little wider than a card's least
// (CardWidthInEms).
const DrawerWidthInEms = 24

/*
Drawer is a surface beside the card stack, in the same window and as tall as
the stack, for what a card has no room to say (spec 068): the processes
behind a rate, a quota in full.

**It is part of the window, not a second one.** A second window would have to
be placed beside the first and kept there through a drag, which on Wayland
only the compositor can do. One window moves as one, and the drawer lines up
with the cards because it is laid out with them.

**It never sets the window's height.** It is given the cards' and draws what
fits: ask Fit how many lines that is. There is still no scroller.

**Opening and closing is a change of shape**, like a card appearing: the
window is resized once, in the same call. The cards do not move on the
screen, on either side.

A panel has one, closed until asked; see Panel.Drawer. What a click means is
the program's: Panel.OnTapped says what was clicked and the program decides
what the drawer shows.
*/
type Drawer struct {
	themed

	panel *Panel
	side  Side
	open  bool

	// fixed is set once a side has been asked for by name. Until then the
	// drawer opens on whichever side of the cards the screen has room on.
	fixed bool

	// held says the compositor keeps the window's right edge where it is
	// when the window is made wider, which is what a left drawer needs where
	// a window cannot move itself. See SetEdgeHeld.
	held bool

	// yield gives the keyboard back where the window cannot. See SetYield.
	yield func()

	title  *canvas.Text
	body   *fyne.Container
	pieces []Piece
	filter *Filter

	face  *canvas.Rectangle
	inner *fyne.Container
	frame *fyne.Container
}

func newDrawer(p *Panel) *Drawer {
	d := &Drawer{panel: p, title: canvas.NewText("", nil)}
	d.title.TextStyle = fyne.TextStyle{Bold: true}
	d.title.Color = d.colour(theme.ColorNamePlaceHolder)
	d.title.TextSize = d.textSize()
	d.body = container.NewVBox()

	// A card's surface: the same fill, border, radius and margins, so the
	// drawer reads as one more card and not as another kind of thing.
	d.face = canvas.NewRectangle(d.colour(theme.ColorNameButton))
	d.face.StrokeColor = d.colour(theme.ColorNameSeparator)
	d.face.StrokeWidth = 1
	d.face.CornerRadius = CardRadius
	d.inner = container.New(d.padding(), container.NewVBox(d.title, d.body))
	d.frame = container.NewStack(d.face, d.inner)
	d.frame.Hide()
	return d
}

func (d *Drawer) padding() fyne.Layout {
	return layout.NewCustomPaddedLayout(d.padTop(), d.padBottom(), d.padH(), d.padH())
}

// width is the drawer's own width.
func (d *Drawer) width() float32 { return d.textSize() * DrawerWidthInEms }

// need is what the drawer adds to the window when it is open: its width and
// the gap between it and the cards. extra is what it adds now.
func (d *Drawer) need() float32 { return d.width() + d.panel.gap() }

func (d *Drawer) extra() float32 {
	if !d.open {
		return 0
	}
	return d.need()
}

// SetTitle names what the drawer is showing, in a card's heading.
func (d *Drawer) SetTitle(s string) {
	if d.title.Text == s {
		return
	}
	d.title.Text = s
	d.title.Refresh()
	d.refit(d.title)
}

// Title is the drawer's heading.
func (d *Drawer) Title() string { return d.title.Text }

// SetContent replaces what the drawer draws under its heading: rows, meters,
// a Filter, in the order given. Each takes the panel's theme and keeps it in
// step, as a card's pieces do.
//
// Replace the content when its shape changes, a list for a set of meters,
// and not when a number does: set the rows in place for that.
func (d *Drawer) SetContent(pieces ...Piece) {
	d.pieces = pieces
	objects := make([]fyne.CanvasObject, len(pieces))
	for i, p := range pieces {
		p.SetTheme(d.th)
		objects[i] = p.Object()
	}
	d.body.Objects = objects
	d.body.Refresh()
}

// Available is the room under the heading, inside the drawer's margins, at
// the height the cards have now.
func (d *Drawer) Available() fyne.Size {
	s := d.panel.cardsSize()
	height := s.Height - d.padTop() - d.padBottom() - d.title.MinSize().Height - d.size(theme.SizeNamePadding)
	return fyne.NewSize(d.width()-2*d.padH(), max(0, height))
}

/*
Fit is how many lines like line fit in the drawer under the pieces named,
which are the ones always shown above the list: a Filter, a note.

It is what a list asks before it draws, because the drawer is the cards'
height and not its own: a list longer than this would run off the bottom of
the window. Ask again when the cards change height, which is when a card
appears or goes.
*/
func (d *Drawer) Fit(line Piece, above ...Piece) int {
	gap := d.size(theme.SizeNamePadding)
	room := d.Available().Height
	for _, p := range above {
		room -= p.Object().MinSize().Height + gap
	}
	each := line.Object().MinSize().Height + gap
	if each <= 0 || room <= 0 {
		return 0
	}
	// The last line has no gap under it.
	return int((room + gap) / each)
}

// Open reports whether the drawer is out.
func (d *Drawer) Open() bool { return d.open }

// Side is the side the drawer is on, or was last on.
func (d *Drawer) Side() Side { return d.side }

// Auto reports whether the side is chosen by the room on the screen.
func (d *Drawer) Auto() bool { return !d.fixed }

// SetAuto goes back to choosing the side by the room on the screen, each
// time the drawer opens. It is what a new drawer does.
func (d *Drawer) SetAuto() { d.fixed = false }

// SetSide puts the drawer on a side of the cards and keeps it there.
func (d *Drawer) SetSide(s Side) {
	d.fixed = true
	d.setSide(s)
}

func (d *Drawer) setSide(s Side) {
	if d.side == s {
		return
	}
	if !d.open {
		d.side = s
		return
	}
	// As far as the window's place is concerned the drawer is closed on one
	// side and opened on the other: what a left drawer added on the left
	// comes off, then goes on where the drawer now is.
	extra := d.extra()
	d.panel.shift(extra, 0)
	d.side = s
	if s == Left && !d.growsLeft() {
		d.side = Right
	}
	d.panel.shift(0, extra)
	d.panel.body.Refresh()
}

/*
Choose picks the side for a drawer that chooses by room, from the room
between the window and each edge of its screen's work area, and returns the
side the drawer will open on. A drawer with a named side keeps it.

The right if the drawer fits there, the left if it fits only there, and the
side with more room if it fits on neither.

On Windows the drawer reads the room itself at each opening and a program has
no need of this. Elsewhere a program cannot read its window's place (quirk
33) and asks the compositor: on Plasma kwin.ReportRoomScript answers in a
millisecond or two. Call it with the answer, then SetOpen. A drawer nobody
told opens on the right.
*/
func (d *Drawer) Choose(left, right float32) Side {
	if d.fixed {
		return d.side
	}
	d.setSide(d.roomiest(left, right))
	return d.side
}

func (d *Drawer) roomiest(left, right float32) Side {
	need := d.need()
	switch {
	case right >= need:
		return Right
	case left >= need, left > right:
		return Left
	default:
		return Right
	}
}

/*
SetEdgeHeld says the compositor keeps the window's right edge where it is
when the window is made wider or narrower.

A left drawer needs it: the window has to grow leftwards for the cards to
stay where they are. On Windows the window moves itself and this is not
needed. On Wayland only the compositor moves a window; on Plasma
kwin.HoldRightEdgeScript, loaded while the drawer is on the left, is what
holds the edge, and the program says so here.

**A drawer that cannot grow leftwards opens on the right**, whatever side was
asked for. The other choice is cards that jump sideways by the drawer's width
under the pointer that clicked them.
*/
func (d *Drawer) SetEdgeHeld(held bool) { d.held = held }

// growsLeft reports whether a left drawer can be opened without the cards
// moving.
func (d *Drawer) growsLeft() bool { return d.panel.selfMoving || d.held }

// SetOpen pulls the drawer out or pushes it back. It is a change of shape:
// the window is resized once, here.
//
// Closing gives the keyboard back (YieldFocus) and ends a filter that was
// being typed.
func (d *Drawer) SetOpen(open bool) {
	if d.open == open {
		return
	}
	before := d.extra()
	if open {
		d.settle()
		d.frame.Show()
	} else {
		d.frame.Hide()
		if d.filter != nil {
			d.filter.stop()
		}
	}
	d.open = open
	d.panel.reshape(before)
	if !open {
		d.YieldFocus()
	}
}

// settle decides the side the drawer is about to open on.
func (d *Drawer) settle() {
	if !d.fixed && d.panel.win != nil {
		if left, right, ok := roomBeside(d.panel.win); ok {
			d.side = d.roomiest(left, right)
		}
	}
	if d.side == Left && !d.growsLeft() {
		d.side = Right
	}
}

/*
SetYield is what gives the keyboard back where the window cannot do it
itself.

On Windows the drawer hands the foreground to the window that had it when the
panel was clicked. A Wayland client cannot activate another program's window:
hiding and showing the panel gives the keyboard away for three milliseconds
and takes it back, and loses the window's place (quirk 49); a request for
the focus is refused. The compositor can, and on Plasma kwin.YieldFocusScript
is the asking. The function given here runs it; it is called on the UI thread
and must not block.
*/
func (d *Drawer) SetYield(yield func()) { d.yield = yield }

/*
YieldFocus gives the keyboard back to the window the person was in before a
click on the panel took it.

A panel is read and not typed into, and the click that opens the drawer makes
the panel the active window: what the person types next goes to it and
nowhere else. Closing the drawer does this, and a Filter does it when Return
or Escape ends it.

It acts only when the panel holds the keyboard, and never brings another
window forward otherwise.
*/
func (d *Drawer) YieldFocus() {
	if d.panel.win == nil {
		return
	}
	if yieldFocus(d.panel.win) {
		return
	}
	if d.yield != nil {
		d.yield()
	}
}

/*
SetFilter gives the drawer a filter and the keys to work it: while the drawer
is open and the panel holds the keyboard, the filter's key starts it, what is
typed narrows the list, Return keeps it and Escape clears it, and either gives
the keyboard back.

The filter is drawn where the program put it in SetContent; this is only the
wiring. It takes the canvas's typed-key and typed-rune handlers, which a
glance window has no other use for: nothing in one holds the focus.

A program whose window rule refuses the focus (kwin.Rule.NoFocus) cannot have
a filter: the panel is never the active window and no key reaches it.
*/
func (d *Drawer) SetFilter(f *Filter) {
	d.filter = f
	if f == nil {
		return
	}
	f.done = d.YieldFocus
	d.wireKeys()
}

// wireKeys points the window's keys at the filter. It is called again when
// the panel is attached to a window, which may be after SetFilter.
func (d *Drawer) wireKeys() {
	if d.filter == nil || d.panel.win == nil {
		return
	}
	cv := d.panel.win.Canvas()
	cv.SetOnTypedRune(func(r rune) {
		if d.open {
			d.filter.TypedRune(r)
		}
	})
	cv.SetOnTypedKey(func(e *fyne.KeyEvent) {
		if d.open {
			d.filter.TypedKey(e)
		}
	})
}

// SetTheme gives the drawer and what it holds a theme of its own. A panel
// calls it; there is nothing for a consumer to do.
func (d *Drawer) SetTheme(th fyne.Theme) {
	d.themed.SetTheme(th)
	for _, p := range d.pieces {
		p.SetTheme(th)
	}
	d.Restyle()
}

// Restyle repaints the drawer's surface and heading in the current theme. Its
// width is a factor of the text size, so the panel resizes after it.
func (d *Drawer) Restyle() {
	d.face.FillColor = d.colour(theme.ColorNameButton)
	d.face.StrokeColor = d.colour(theme.ColorNameSeparator)
	d.face.CornerRadius = CardRadius
	d.face.Refresh()
	d.title.TextSize = d.textSize()
	d.title.Color = d.colour(theme.ColorNamePlaceHolder)
	d.title.Refresh()
	d.refit(d.title)
	d.inner.Layout = d.padding()
	d.inner.Refresh()
}

// drawerLayout puts the card stack and the drawer side by side. The height is
// the cards' alone.
type drawerLayout struct{ panel *Panel }

func (l *drawerLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	s := l.panel.stack.MinSize()
	s.Width += l.panel.drawer.extra()
	return s
}

func (l *drawerLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	p, d := l.panel, l.panel.drawer
	extra := d.extra()
	cards := fyne.NewSize(size.Width-extra, size.Height)
	origin := fyne.NewPos(0, 0)
	if extra > 0 {
		dw := d.width()
		if d.side == Left {
			d.frame.Move(fyne.NewPos(0, 0))
			origin.X = extra
		} else {
			d.frame.Move(fyne.NewPos(size.Width-dw, 0))
		}
		d.frame.Resize(fyne.NewSize(dw, size.Height))
	}
	p.stack.Move(origin)
	p.stack.Resize(cards)
}

// Drawer is the panel's drawer. It is there from the start, closed and
// empty.
func (p *Panel) Drawer() *Drawer { return p.drawer }

// cardsSize is the size of the card stack alone: the content's minimum,
// widened to the panel's floor.
func (p *Panel) cardsSize() fyne.Size {
	s := p.stack.MinSize()
	if s.Width < p.minWidth {
		s.Width = p.minWidth
	}
	return s
}

// reshape resizes the window after the drawer's width changed from before,
// moving the window's origin when the drawer is on the left.
func (p *Panel) reshape(before float32) {
	after := p.drawer.extra()
	p.body.Refresh()
	if p.win == nil {
		return
	}
	// Growing leftwards moves before it grows and shrinking moves after, so
	// the cards are never drawn outside the window.
	if after > before {
		p.shift(before, after)
		p.Resize()
		return
	}
	p.Resize()
	p.shift(before, after)
}

// shift moves the window's origin for a left drawer whose width went from
// before to after.
func (p *Panel) shift(before, after float32) {
	if p.win == nil || p.drawer.side != Left || before == after {
		return
	}
	if p.selfMoving {
		moveWindowBy(p.win, before-after)
		return
	}
	// The compositor moved the window, and the pointer was not told (quirk
	// 48): the cards are now further from the window's left edge by this
	// much, under a pointer Fyne believes is where it was. Fyne is told
	// where it is; where it cannot be told, the next click is read with the
	// difference added (Panel.tapped).
	if !repoint(p.win, &p.pointer, after-before) {
		p.stale += after - before
	}
}

// pointer is what repoint remembers between two moves of the window: where
// GLFW last said the pointer was, and how far the window has moved under it
// since.
type pointer struct {
	x, y  float64
	heard bool
	moved float32

	// watched is set once the window's focus is being watched, to say the
	// place again when Fyne forgets it.
	watched bool
}
