package glance

import (
	"time"

	"fyne.io/fyne/v2"
)

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
func (s *Sparkline) Ticking() bool { return s.stop != nil }
