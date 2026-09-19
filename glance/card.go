package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// GoneMarker is what a card's header says when its source was answering and
// has stopped. It is short because it shares the header with the title, and it
// is a state rather than a mechanism: the reason belongs in the log.
const GoneMarker = "(unavailable)"

// Card is one metric's panel: a title, some rows, and optionally something
// drawn under them such as a Sparkline.
//
// A card is drawn only when the user allows it and its source has something to
// say. The two are separate on purpose — a card the user hid and a card with
// no data look the same and arrive by different paths, and a program that
// conflated them would bring a hidden card back the moment its sensor
// appeared.
//
// A card that is not drawn costs nothing. Stop its timer in the callback
// handed to OnDrawnChanged: a glance window on a machine without the sensor it
// watches should be indistinguishable from one built without the card.
type Card struct {
	title *canvas.Text
	mark  *canvas.Text
	rows  []*Row
	body  *fyne.Container
	frame *fyne.Container

	allowed   bool
	available bool
	stale     bool
	drawn     bool

	// OnDrawnChanged is called when the card starts or stops being drawn, so
	// the owner can start and stop its poll. It is called on the UI thread.
	OnDrawnChanged func(drawn bool)
}

// NewCard builds a card with a title and no rows. It is allowed by default and
// not available: a card is drawn once its source answers, so a window does not
// flash an empty panel on the way up.
func NewCard(title string) *Card {
	c := &Card{
		title:   canvas.NewText(title, theme.Color(theme.ColorNameForeground)),
		mark:    canvas.NewText(GoneMarker, theme.Color(theme.ColorNameDisabled)),
		allowed: true,
	}
	c.title.TextStyle = fyne.TextStyle{Bold: true}
	c.title.TextSize = theme.TextSize()
	c.mark.TextSize = theme.TextSize()
	c.mark.Hide()

	header := container.NewHBox(c.title, layout.NewSpacer(), c.mark)
	c.body = container.NewVBox()
	c.frame = container.NewVBox(header, c.body)
	c.frame.Hide()
	return c
}

// AddRow appends rows in the order they will be drawn.
func (c *Card) AddRow(rows ...*Row) {
	for _, r := range rows {
		c.rows = append(c.rows, r)
		c.body.Add(r.Object())
	}
}

// AddObject appends something that is not a row — a Sparkline, a progress bar
// — under the rows added so far.
func (c *Card) AddObject(o fyne.CanvasObject) { c.body.Add(o) }

// Rows are the card's rows, in order, for a caller that keeps no handles of
// its own and for tests.
func (c *Card) Rows() []*Row { return c.rows }

// SetAllowed records the user's toggle: whether this card is wanted at all.
// It is the context menu's half of the decision.
func (c *Card) SetAllowed(allowed bool) {
	c.allowed = allowed
	c.apply()
}

// Allowed reports the user's toggle.
func (c *Card) Allowed() bool { return c.allowed }

// SetAvailable records whether the source has anything to say. It is the
// poll's half of the decision: true on the first answer, and left true
// afterwards, because a source that has answered once and stopped is stale
// rather than absent. Use SetStale for that.
func (c *Card) SetAvailable(available bool) {
	c.available = available
	c.apply()
}

// Available reports whether the source has anything to say.
func (c *Card) Available() bool { return c.available }

// SetStale dims every row and marks the header, for a source that was
// answering and has stopped. The card keeps its last values: a reader's
// question is whether they are still true, and a dim number answers it without
// moving anything. Recovery clears the marker.
func (c *Card) SetStale(stale bool) {
	if c.stale == stale {
		return
	}
	c.stale = stale
	for _, r := range c.rows {
		rd := r.Reading()
		rd.Stale = stale
		r.Set(rd)
	}
	if stale {
		c.mark.Show()
	} else {
		c.mark.Hide()
	}
}

// Stale reports whether the card is showing last-known values.
func (c *Card) Stale() bool { return c.stale }

// Drawn reports whether the card is on screen: allowed by the user and
// available from its source.
func (c *Card) Drawn() bool { return c.drawn }

// apply reconciles the two halves and tells the owner when the answer changes.
func (c *Card) apply() {
	drawn := c.allowed && c.available
	if drawn == c.drawn {
		return
	}
	c.drawn = drawn
	if drawn {
		c.frame.Show()
	} else {
		c.frame.Hide()
	}
	if c.OnDrawnChanged != nil {
		c.OnDrawnChanged(drawn)
	}
}

// Restyle repaints the card and its rows in the current theme.
func (c *Card) Restyle() {
	c.title.TextSize = theme.TextSize()
	c.title.Color = theme.Color(theme.ColorNameForeground)
	c.mark.TextSize = theme.TextSize()
	c.mark.Color = theme.Color(theme.ColorNameDisabled)
	c.title.Refresh()
	c.mark.Refresh()
	for _, r := range c.rows {
		r.Restyle()
	}
}

// Object is the card's content, for a caller assembling a panel by hand.
func (c *Card) Object() fyne.CanvasObject { return c.frame }
