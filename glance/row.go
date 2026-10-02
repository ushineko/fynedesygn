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

	// labelBox pins the label column, for a row built by NewRowWidth. Nil
	// for a row whose label takes its own width.
	labelBox *fyne.Container

	// slot holds the value: the plain text first, then one text per part.
	// The plain text is always there and always carries the whole value,
	// because it is what the slot is measured by -- hidden while the parts
	// are drawn, so a parted value and the same text unparted are one width
	// by construction rather than by arithmetic.
	slot  *fyne.Container
	parts []*canvas.Text

	reading Reading
	shown   bool
}

// NewRow builds a row with a label and no value yet. The value reads as blank
// until Set is called, at the width blank implies, so the row does not change
// size when its first reading lands.
func NewRow(label, blank string) *Row {
	return NewRowWidth(label, blank, 0)
}

/*
NewRowWidth builds a row whose label column is labelWidth pixels wide,
whatever the label says. 0 is NewRow.

For a row whose label arrives later than the card: a hardware name read from
the device ("Kraken Elite V2", "i7-13700K") widened its card from 194 to 230
px on one desk when it replaced the placeholder (hayami spec 031). Pinned, the
name cannot move the card.

**A label longer than the width is clipped**, not wrapped: the text is given
exactly the width to draw in, and a canvas.Text clips what does not fit (quirk
39). Truncate with an ellipsis before handing it over if it can be long.

This is a pin, not the floor widgets.FixedWidth is. FixedWidth stacks a
spacer under the object and a stack is as wide as its widest child, so a long
label would still have widened the row.
*/
func NewRowWidth(label, blank string, labelWidth float32) *Row {
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

	var head fyne.CanvasObject = r.label
	if labelWidth > 0 {
		r.labelBox = container.New(pinnedWidth(labelWidth), r.label)
		head = r.labelBox
	}
	r.slot = container.New(valueLayout{}, r.value)
	r.box = container.NewHBox(head, layout.NewSpacer(), r.slot)
	r.reading = Measured(blank)
	return r
}

// Set replaces the row's value. It repaints; it does not re-lay-out, because
// the text it is given is the same width as the text it had.
//
// A reading with Parts is drawn as those parts. The part texts are kept from
// one Set to the next while there are as many of them, and the value is laid
// out again only when the count or a part's width changed; the row's size
// does not change either way.
func (r *Row) Set(rd Reading) {
	if len(rd.Parts) > 0 && rd.Text == "" {
		rd.Text = partsText(rd.Parts)
	}
	r.reading = rd
	if len(rd.Parts) == 0 {
		r.setPlain()
		return
	}
	r.setParts()
}

// setPlain draws the value as one text, as a row always did.
func (r *Row) setPlain() {
	r.value.Text = r.reading.Text
	r.value.Color = r.readingColour()
	relayout := !r.value.Visible()
	for _, t := range r.parts {
		t.Hide()
	}
	r.value.Show()
	r.value.Refresh()
	if relayout {
		r.slot.Refresh()
	}
}

// setParts draws the value as its parts, reusing the texts it has.
func (r *Row) setParts() {
	ps := r.reading.Parts
	relayout := r.value.Visible() || len(ps) != len(r.parts)
	if len(ps) != len(r.parts) {
		r.parts = r.parts[:0]
		for range ps {
			r.parts = append(r.parts, canvas.NewText("", nil))
		}
		objs := make([]fyne.CanvasObject, 0, len(ps)+1)
		objs = append(objs, r.value)
		for _, t := range r.parts {
			objs = append(objs, t)
		}
		r.slot.Objects = objs
	}

	// The plain text carries the whole value even while hidden: it is what
	// the slot is measured by.
	r.value.Text = partsText(ps)
	r.value.Hide()
	for i, t := range r.parts {
		r.paintPart(t, ps[i])
		t.Show()
		if t.Size().Width != t.MinSize().Width {
			relayout = true
		}
	}
	if relayout {
		r.slot.Refresh()
		return
	}
	for _, t := range r.parts {
		t.Refresh()
	}
}

// paintPart gives one part text its words, face and colour.
func (r *Row) paintPart(t *canvas.Text, p Part) {
	t.Text = p.Text
	t.TextSize = r.textSize()
	style := fyne.TextStyle{Monospace: true}
	if p.Bold {
		style = r.bold(style)
	}
	t.TextStyle = style
	t.FontSource = r.Theme().Font(style)
	t.Color = r.partColour(p)
}

// partColour is readingColour for one part: Stale is the reading's, the rest
// the part's own.
func (r *Row) partColour(p Part) color.Color {
	if r.reading.Stale {
		return r.colour(theme.ColorNameDisabled)
	}
	if p.Colour != nil {
		return p.Colour
	}
	if p.Status == fd.StatusInfo {
		return r.colour(theme.ColorNameForeground)
	}
	return r.statusColour(p.Status)
}

// SetLabel replaces the row's label, for a row whose subject can change (a
// slot showing whichever device is connected). A row built by NewRowWidth
// keeps its size whatever the label says.
func (r *Row) SetLabel(s string) {
	r.label.Text = s
	r.label.Refresh()
	if r.labelBox != nil {
		r.labelBox.Refresh()
	}
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
	for i, t := range r.parts {
		if i < len(r.reading.Parts) {
			r.paintPart(t, r.reading.Parts[i])
		}
	}
	r.refit(append([]*canvas.Text{r.label, r.value}, r.parts...)...)
	// The box's Refresh lays out every container in it, the label's pin and
	// the value's slot included, which puts the parts end to end again at
	// their new widths and gives a pinned label its width back after refit
	// measured it at its own.
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

// pinnedWidth is a layout exactly w wide, whatever it holds. Its objects are
// given that width to draw in, so a text longer than it is clipped (quirk 39)
// rather than widening the row.
type pinnedWidth float32

func (w pinnedWidth) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var h float32
	for _, o := range objects {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(float32(w), h)
}

func (w pinnedWidth) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(float32(w), size.Height))
	}
}

// valueLayout is a row's value slot: the plain text, which it is measured by
// whether or not it is shown, then the parts laid end to end at their own
// widths.
//
// The parts are not in an HBox because an HBox puts the theme's padding
// between them, and a parted value would then be wider than its text.
type valueLayout struct{}

func (valueLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.Size{}
	}
	return objects[0].MinSize()
}

func (valueLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
	var x float32
	for _, o := range objects[1:] {
		w := o.MinSize().Width
		o.Move(fyne.NewPos(x, 0))
		o.Resize(fyne.NewSize(w, size.Height))
		x += w
	}
}
