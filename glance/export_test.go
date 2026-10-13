package glance

import "fyne.io/fyne/v2"

// SetSelfMoving makes a panel behave as on a desktop where a window can, or
// cannot, move itself, so both are tested on one system.
func (p *Panel) SetSelfMoving(v bool) { p.selfMoving = v }

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
