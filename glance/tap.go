package glance

import "fyne.io/fyne/v2"

/*
Tap is what a primary click on a panel landed on: the card, and the row or the
meter under the pointer when there was one.

A glance window is read and not operated, and almost nothing in one is a
control. A click is the exception a drawer needs: it says which reading the
person wants more of. The library reports it and decides nothing; whether a
click opens a drawer, changes what it shows or closes it is the program's.
*/
type Tap struct {
	// Card is the card clicked. Never nil.
	Card *Card
	// Row is the row under the pointer, or nil: a click on the card's
	// heading, its plot or its margin.
	Row *Row
	// Meter is the meter under the pointer, or nil.
	Meter *Meter
}

// tapped finds what a primary tap at a canvas position landed on and tells
// the program.
func (p *Panel) tapped(at fyne.Position) {
	// A click that arrives where the last one did, after the compositor
	// moved the window under a pointer that has not moved, is read at the old
	// place (quirk 48). The window's content has moved under it by stale.
	// This is the fallback: where Fyne's own idea of the pointer can be put
	// right (repoint), stale stays at nothing.
	if at != p.staleAt {
		p.stale = 0
	}
	p.staleAt = at
	at.X += p.stale

	if p.OnTapped == nil {
		return
	}
	drv := fyne.CurrentApp().Driver()
	inside := func(o fyne.CanvasObject) bool {
		if !o.Visible() {
			return false
		}
		pos, size := drv.AbsolutePositionForObject(o), o.Size()
		return at.X >= pos.X && at.X < pos.X+size.Width && at.Y >= pos.Y && at.Y < pos.Y+size.Height
	}
	for _, c := range p.cards {
		if !inside(c.Object()) {
			continue
		}
		tap := Tap{Card: c}
		for _, r := range c.rows {
			if r.Shown() && inside(r.Object()) {
				tap.Row = r
				break
			}
		}
		for _, m := range c.meters {
			if tap.Row == nil && inside(m.Object()) {
				tap.Meter = m
				break
			}
		}
		p.OnTapped(tap)
		return
	}
}
