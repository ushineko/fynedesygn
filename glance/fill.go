package glance

import (
	"math"

	"fyne.io/fyne/v2"
)

/*
A full row of drawers fills the room left beside it (spec 080).

The row stops adding drawers when one more would not fit (Fits), and stops
short of the screen's edge by anything up to a drawer. Filling widens every
open drawer by the same amount until the window reaches the edge.

**The widths are worked out in device pixels.** A drawer is not a whole
number of pixels wide at a scale of 1.5, and three drawers placed by Fyne's
units round to widths a pixel apart. So the window's width in pixels is
decided first, the cards' and the gaps' pixels are taken from it, and what is
left is divided by the drawers: each gets the same whole number of pixels,
and the few the division leaves over stand beyond the outermost drawer as
padding, the panel's background.
*/

// filling is the row's widths while it is filled, in device pixels at
// scale. cards is the cards' share of the window, gap a gap's, widths each
// drawer's in the row's order, and pad what the division left over.
type filling struct {
	scale              float32
	window, cards, gap int
	widths             []int
	pad, drawers       int
	room               float32
}

/*
planFill works out a filled row of n drawers beside cards wide, with drawers
of their own width taking extra and room to spare beyond them, at scale.

The window is as many pixels wide as Fyne makes the window it has now
(ceil, as pixelsMoved reads it, #238) and the whole pixels of the room; a
fraction of a pixel of room is not room.

**A width that is a whole number of pixels is that number**, not one more.
Cards 289.333 wide at 1.5 are 434 pixels, but 289.333 in float32 times 1.5
is 434.00001, and rounded up the cards were planned at 435: on the desk the
cards' edge moved a pixel when the row filled. So the rounding up forgives
what float32 adds.
*/
func planFill(scale, cards, extra, gap, room float32, own []float32) (filling, bool) {
	n := len(own)
	if n <= 0 || scale <= 0 {
		return filling{}, false
	}
	ceil := func(v float32) int { return int(math.Ceil(float64(v*scale) - 1e-3)) }
	spare := int(math.Floor(float64(room*scale) + 1e-3))
	if spare <= 0 {
		return filling{}, false
	}
	f := filling{
		scale:  scale,
		window: ceil(cards+extra) + spare,
		cards:  ceil(cards),
		gap:    int(math.Round(float64(gap * scale))),
		room:   room,
	}
	f.drawers = f.window - f.cards - n*f.gap
	if f.drawers < n {
		return filling{}, false
	}
	// Each drawer keeps its own whole pixels and is given the same share of
	// what is left (spec 082 R8). Drawers of one width are spec 080's row:
	// floor(drawers/n) each.
	pixels := make([]int, n)
	used := 0
	for i, w := range own {
		pixels[i] = int(math.Floor(float64(w*scale) + 1e-3))
		used += pixels[i]
	}
	share := int(math.Floor(float64(f.drawers-used) / float64(n)))
	f.widths = make([]int, n)
	f.pad = f.drawers
	for i, px := range pixels {
		f.widths[i] = px + share
		if f.widths[i] < 1 {
			return filling{}, false
		}
		f.pad -= f.widths[i]
	}
	return f, true
}

// before is the pixels the drawers before the i-th take, each with its gap.
func (f filling) before(i int) int {
	x := 0
	for _, w := range f.widths[:i] {
		x += f.gap + w
	}
	return x
}

// unit is a pixel count in Fyne's units.
func (f filling) unit(px int) float32 { return float32(px) / f.scale }

// extra is what the filled row adds to the cards' width, in units: a hair
// under the pixels so that the window Fyne makes of it, rounding up, is the
// window planned and not a pixel wider.
func (f filling) extra(cards float32) float32 {
	return (float32(f.window)-0.01)/f.scale - cards
}

/*
Fill widens the open drawers until the window reaches the edge of the work
area on their side, given the room between the window and each edge (as Fits
takes it), and reports whether it did. With no drawer open, or no whole pixel
of room on that side, it changes nothing.

Filled again, the room is added to what the row has already taken.

Closing a drawer, opening one, CloseDrawers and Restyle end it. What a drawer
shows does not.
*/
func (p *Panel) Fill(left, right float32) bool {
	n := len(p.row)
	if n == 0 {
		return false
	}
	room := right
	if p.side == Left {
		room = left
	}
	if p.filled {
		room += p.fill.room
	}
	cards := p.CardsSize().Width
	own := make([]float32, n)
	for i, d := range p.row {
		own[i] = d.width()
	}
	f, ok := planFill(p.scale(), cards, p.ownExtra(), p.gap(), room, own)
	if !ok {
		return false
	}
	before := p.drawersExtra()
	p.fill, p.filled = f, true
	p.fillCards = cards
	p.reshape(before)
	p.placeFill()
	return true
}

// FillHere is Fill with the room read by the window itself, where it can
// (Windows). Where it cannot, known is false: ask the compositor and call
// Fill.
func (p *Panel) FillHere() (ok, known bool) {
	if p.win == nil {
		return false, false
	}
	left, right, have := roomBeside(p.win)
	if !have {
		return false, false
	}
	return p.Fill(left, right), true
}

// Unfill puts every open drawer back to its own width.
func (p *Panel) Unfill() {
	if !p.filled {
		return
	}
	before := p.drawersExtra()
	p.filled = false
	p.reshape(before)
	p.placeFill()
}

// Filled reports whether the row is filled.
func (p *Panel) Filled() bool { return p.filled }

// DrawersWidth is what the open drawers add to the window's width now,
// filled or not: the window's width less the cards'. A program that works
// the cards' corner out of the window's width reads it.
func (p *Panel) DrawersWidth() float32 { return p.drawersExtra() }

/*
SetFillable shows the Fill button at the right of the outermost open
drawer's heading, or hides it; a press calls toggle.

The program says when, because the row is full when the screen has no room
for another drawer and only the program can ask the compositor that. The
button follows the outermost drawer as the row changes, and is drawn engaged
while the row is filled.
*/
func (p *Panel) SetFillable(on bool, toggle func()) {
	p.fillable, p.fillToggle = on, toggle
	p.placeFill()
}

// Fillable reports whether the Fill button is offered (SetFillable).
func (p *Panel) Fillable() bool { return p.fillable }

// FillWord is what the Fill button reads.
const FillWord = "Fill"

// placeFill puts the Fill button on the outermost open drawer and takes it
// off every other.
func (p *Panel) placeFill() {
	out := p.Outermost()
	for _, d := range p.drawers {
		show := p.fillable && d == out
		if !show && d.fill == nil {
			continue
		}
		if d.fill == nil {
			d.fill = newDrawerButton(d)
			d.rebuildTitle()
		}
		if !show {
			d.fill.on = nil
			d.fill.box.Hide()
			d.headingChanged()
			continue
		}
		toggle := p.fillToggle
		apply(d.fill, DrawerSpec{Label: FillWord, Words: []string{FillWord}, Active: p.filled, On: func() {
			if toggle != nil {
				toggle()
			}
		}})
		d.headingChanged()
	}
}

// endFill forgets a filled row without resizing, for a caller that resizes
// once for everything it changed.
func (p *Panel) endFill() { p.filled = false }

// ownExtra is what the open drawers add at their own width.
func (p *Panel) ownExtra() float32 {
	var w float32
	for _, d := range p.row {
		w += d.need()
	}
	return w
}

// scale is the window's scale, or 1 without a window.
func (p *Panel) scale() float32 {
	if p.win != nil {
		if s := p.win.Canvas().Scale(); s > 0 {
			return s
		}
	}
	return 1
}

/*
placeFilled stands the i-th drawer of a filled row at whole pixels, so that
every drawer is drawn the same number of pixels wide: on the right counted
out from the cards, on the left counted in from the window's right edge,
and the padding at the far end either way. shift is the layout's, for a
window put back to a width it just left (putBack).
*/
func (p *Panel) placeFilled(d *Drawer, i int, shift, height float32) {
	f := p.fill
	x := f.cards + f.gap + f.before(i)
	if p.side == Left {
		x = f.window - f.cards - f.before(i+1)
	}
	d.frame.Move(fyne.NewPos(shift+f.unit(x), 0))
	d.frame.Resize(fyne.NewSize(f.unit(f.widths[i]), height))
}
