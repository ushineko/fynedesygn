package glance

import (
	"time"

	"fyne.io/fyne/v2"
)

// PixelsMoved is pixelsMoved, for the test of the right edge on Windows.
func PixelsMoved(scale, before, after float32) int32 { return pixelsMoved(scale, before, after) }

// SetSelfMoving makes a panel behave as on a desktop where a window can, or
// cannot, move itself, so both are tested on one system.
func (p *Panel) SetSelfMoving(v bool) { p.selfMoving = v }

// CardsWidth is the width the card stack is drawn at.
func (p *Panel) CardsWidth() float32 { return p.stack.Size().Width }

// HoldLater keeps what the panel would run after a layout pass and returns
// the function that runs it, so a test sees the panel in between.
func (p *Panel) HoldLater() (run func()) {
	var held []func()
	p.later = func(f func()) { held = append(held, f) }
	return func() {
		for _, f := range held {
			f()
		}
		held = nil
	}
}

// SetShift makes every click after it one with Shift held, or not: the test
// driver keeps no keyboard state.
func (p *Panel) SetShift(held bool) { p.shifted = func() bool { return held } }

// CountResizes counts the window's resizes from now on.
func (p *Panel) CountResizes() *int {
	n, was := 0, p.hook
	p.hook = func() {
		n++
		if was != nil {
			was()
		}
	}
	return &n
}

// CardsOrigin is where the card stack is drawn in the window.
func (p *Panel) CardsOrigin() fyne.Position { return p.stack.Position() }

// Frame is the drawer's surface.
func (d *Drawer) Frame() fyne.CanvasObject { return d.frame }

// Line is what the filter's line says.
func (f *Filter) Line() string { return f.line.Text }

// Catcher is the object stacked over the panel that takes its pointer events.
func (w *Window) Catcher() fyne.CanvasObject {
	return w.panel.root.Objects[len(w.panel.root.Objects)-1]
}

// ChoiceObjects are the switch's choices as drawn, in order.
func (s *Switch) ChoiceObjects() []fyne.CanvasObject { return s.box.Objects }

// SetClock replaces the sparkline's clock, so a test draws a drift at the
// moment it chooses.
func (s *Sparkline) SetClock(now func() time.Time) { s.now = now }

// HoldDriftTicks stops drifting plots starting their redraws, so nothing but
// the test moves them, and returns what puts it back.
func HoldDriftTicks() (restore func()) {
	driftTicks = false
	return func() { driftTicks = true }
}

// Ticking reports whether the plot's redraws are running.
func (s *Sparkline) Ticking() bool { return s.anim != nil }

// Head is the row's label column: the pinned container for a row built by
// NewRowWidth, and the bare label otherwise.
//
// A test asking "is the column still the width it was pinned to" wants that
// object and not the row's first child, which is the ground a hover is drawn
// on and is as wide as the row.
func (r *Row) Head() fyne.CanvasObject {
	if r.labelBox != nil {
		return r.labelBox
	}
	return r.label
}

// PlanFill is planFill's pixels: each drawer's, the padding's, the window's.
func PlanFill(scale, cards, extra, gap, room float32, n int) (each, pad, window int, ok bool) {
	own := make([]float32, n)
	for i := range own {
		own[i] = extra/float32(n) - gap
	}
	f, ok := planFill(scale, cards, extra, gap, room, own)
	if !ok {
		return 0, 0, 0, false
	}
	return f.widths[0], f.pad, f.window, ok
}

// PlanFillOwn is planFill's pixels for drawers of the widths given.
func PlanFillOwn(scale, cards, gap, room float32, own []float32) (widths []int, pad, window int, ok bool) {
	var extra float32
	for _, w := range own {
		extra += w + gap
	}
	f, ok := planFill(scale, cards, extra, gap, room, own)
	return f.widths, f.pad, f.window, ok
}

// Gap is the gap between the cards and a drawer, and between drawers.
func (p *Panel) Gap() float32 { return p.gap() }

// FillButton is the drawer's Fill button's label, and whether it is shown
// and drawn engaged; "" when the drawer has none.
func (d *Drawer) FillButton() (label string, shown, active bool) {
	if d.fill == nil {
		return "", false, false
	}
	return d.fill.text.Text, d.fill.box.Visible(), d.fill.active
}

// TuckSquare is the tucked square, and TuckGlyph the glyph that asks to
// tuck.
func (p *Panel) TuckSquare() fyne.CanvasObject { return p.tuck.square }
func (p *Panel) TuckGlyph() fyne.CanvasObject  { return p.tuck.glyph }

// Body is the cards and drawers together.
func (p *Panel) Body() fyne.CanvasObject { return p.body }

// PressFill presses the drawer's Fill button.
func (d *Drawer) PressFill() { d.fill.on() }

// Offset is how far a scroller's rows are scrolled, in pixels.
func (s *Scroller) Offset() float32 { return s.scroll.Offset.Y }
