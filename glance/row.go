package glance

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	fd "github.com/ushineko/fynedesygn"
)

// Row is one label/value line in a card: the label on the left, the value on
// the right, with the space between them taking up the slack.
//
// The value is drawn in the monospace face. Padding the text to a fixed width
// (see format.go) only keeps the column still if the digits are the same width
// as each other, which in a proportional face they are not.
//
// A Row is a live handle, not a build function. Cards are redrawn from a poll
// several times a minute and rebuilding the window at that rate would fight
// the card's own no-reflow rule; this is the one place a glance window departs
// from the design system's "rebuild the section from state".
type Row struct {
	themed

	label *canvas.Text
	value *canvas.Text
	box   *fyne.Container

	reading Reading
	shown   bool
}

// NewRow builds a row with a label and no value yet. The value reads as blank
// until Set is called, at the width blank implies, so the row does not change
// size when its first reading lands.
func NewRow(label, blank string) *Row {
	r := &Row{
		label: canvas.NewText(label, nil),
		value: canvas.NewText(blank, nil),
		shown: true,
	}
	r.label.Color = r.colour(theme.ColorNameForeground)
	r.value.Color = r.colour(theme.ColorNameForeground)
	r.label.TextSize = r.textSize()
	r.value.TextSize = r.textSize()
	r.value.TextStyle = fyne.TextStyle{Monospace: true}
	r.box = container.NewHBox(r.label, layout.NewSpacer(), r.value)
	r.reading = Measured(blank)
	return r
}

// Set replaces the row's value. It repaints; it does not re-lay-out, because
// the text it is given is the same width as the text it had.
func (r *Row) Set(rd Reading) {
	r.reading = rd
	r.value.Text = rd.Text
	r.value.Color = r.readingColour()
	r.value.Refresh()
}

// SetLabel replaces the row's label, for a row whose subject can change (a
// slot showing whichever device is connected).
func (r *Row) SetLabel(s string) {
	r.label.Text = s
	r.label.Refresh()
}

// SetTheme gives the row a face of its own and repaints in it. A canvas.Text
// holds a literal size and colour rather than asking the theme when it draws,
// so being told is not enough.
func (r *Row) SetTheme(th fyne.Theme) {
	r.themed.SetTheme(th)
	r.Restyle()
}

// Reading is the value the row currently holds, which is what a test asks for
// rather than walking the object tree.
func (r *Row) Reading() Reading { return r.reading }

// colour is the status colour, or the disabled colour when the value is stale.
// Dimming wins over the verdict: a warning that is no longer being refreshed
// should not keep shouting.
func (r *Row) readingColour() color.Color {
	if r.reading.Stale {
		return r.colour(theme.ColorNameDisabled)
	}
	if r.reading.Colour != nil {
		return r.reading.Colour
	}
	if r.reading.Status == fd.StatusInfo {
		return r.colour(theme.ColorNameForeground)
	}
	return r.statusColour(r.reading.Status)
}

// Restyle repaints the row in the current theme, after a scheme or text size
// change. A canvas.Text holds a literal colour and size rather than asking the
// theme at paint time, so nothing else brings it up to date.
func (r *Row) Restyle() {
	r.label.TextSize = r.textSize()
	r.value.TextSize = r.textSize()
	r.label.Color = r.colour(theme.ColorNameForeground)
	r.value.Color = r.readingColour()
	r.label.Refresh()
	r.value.Refresh()
	r.refit(r.label, r.value)
	r.box.Refresh()
}

// SetShown draws or hides the row. A metric its source cannot read hides its
// own row and leaves the card standing for the rows that do have data.
func (r *Row) SetShown(shown bool) {
	if r.shown == shown {
		return
	}
	r.shown = shown
	if shown {
		r.box.Show()
	} else {
		r.box.Hide()
	}
}

// Shown reports whether the row is currently drawn.
func (r *Row) Shown() bool { return r.shown }

// Object is the row's content, for a caller assembling a card by hand.
func (r *Row) Object() fyne.CanvasObject { return r.box }
