package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

/*
The drawer's button and its hover (spec 063).

A drawer was a page to read, so the only thing on it that answered to a
pointer was the switch. A page a person works -- stops, walks into, comes
back out of -- has to show what can be worked: a control that looks like a
control, and, under a pointer, which row it is over. At the text sizes a
panel is read at, a row is four millimetres tall and "which row am I on" is
not a question a person should have to answer by aiming.
*/

// drawerButton is a label on a filled, rounded ground, at the right of the
// drawer's heading: a thing that looks pressable, because a bare word in a
// list does not.
type drawerButton struct {
	face  *canvas.Rectangle
	text  *canvas.Text
	box   *fyne.Container
	on    func()
	owner *Drawer

	// active is whether it is drawn as an engaged toggle.
	active bool

	// widest is the width the button is held at whatever it says, so a
	// control that changes its word does not change its shape under the
	// pointer that is about to press it.
	widest float32
}

func newDrawerButton(d *Drawer) *drawerButton {
	b := &drawerButton{owner: d}
	// Smaller than the heading beside it, and lit rather than sunk: an
	// input's ground is a hole to type into, and this is a thing to press.
	b.text = canvas.NewText("", d.colour(theme.ColorNameForeground))
	b.text.TextSize = d.textSize() * buttonTextScale
	b.face = canvas.NewRectangle(d.colour(theme.ColorNameHover))
	b.face.StrokeColor = d.colour(theme.ColorNameSeparator)
	b.face.StrokeWidth = 1
	d.refit(b.text)
	b.box = container.NewStack(b.face, container.New(buttonPad{d: d, b: b}, b.text))
	b.box.Hide()
	return b
}

// buttonPad gives the label a little room inside its ground.
type buttonPad struct {
	d *Drawer
	b *drawerButton
}

func (p buttonPad) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.Size{}
	}
	s := objects[0].MinSize()
	pad := p.d.size(theme.SizeNamePadding)
	// Room at the ends and above: a pill whose word touches its edge reads
	// as a word with a box drawn round it rather than as a thing to press.
	width := s.Width
	if p.b != nil && p.b.widest > width {
		width = p.b.widest
	}
	return fyne.NewSize(width+pad*4, s.Height+buttonTop(pad)+buttonBottom(pad))
}

/*
buttonTop and buttonBottom are the room above and below the word.

They are not equal. A text's box is as tall as the face it is set in --
ascender to descender -- and a word like "Pause" or "Back" has nothing
below the baseline, so its ink sits high in that box and the button reads
as bottom-heavy when the box is centred. The ink is centred instead, by
giving the box more room beneath than above.
*/
func buttonTop(pad float32) float32 { return pad * 0.8 }

func buttonBottom(pad float32) float32 { return pad * 2.2 }

// buttonTextScale is how much smaller the button's word is than the text
// around it: it is a control at the edge of a heading, not a heading.
const buttonTextScale = 0.9

func (p buttonPad) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	s := objects[0].MinSize()
	pad := p.d.size(theme.SizeNamePadding)
	objects[0].Resize(s)
	// Across the middle, and down from the top by the lesser of the two
	// paddings, which is what puts the ink in the middle rather than the
	// box it is measured in.
	objects[0].Move(fyne.NewPos((size.Width-s.Width)/2, buttonTop(pad)))
}

// buttonAt reports whether a position is on the button, and presses it.
func (d *Drawer) buttonAt(at fyne.Position) bool {
	for _, b := range append([]*drawerButton{d.lead}, d.buttons...) {
		if b == nil || b.on == nil || !b.box.Visible() {
			continue
		}
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(b.box)
		size := b.box.Size()
		if at.X < pos.X || at.X >= pos.X+size.Width ||
			at.Y < pos.Y || at.Y >= pos.Y+size.Height {
			continue
		}
		b.on()
		return true
	}
	return false
}

/*
restyle repaints the button in the current theme.

The size, the colours and the face all follow the theme, and the face is the
reason this exists: a text takes its font from the theme only when it is
told to (themed.refit), and a button built once and never restyled kept the
font it was born with while every row around it changed.
*/
func (b *drawerButton) restyle(d *Drawer) {
	b.text.TextSize = d.textSize() * buttonTextScale
	if b.active {
		b.face.FillColor = d.colour(theme.ColorNamePrimary)
		b.text.Color = d.colour(theme.ColorNameForegroundOnPrimary)
	} else {
		b.face.FillColor = d.colour(theme.ColorNameHover)
		b.text.Color = d.colour(theme.ColorNameForeground)
	}
	b.face.StrokeColor = d.colour(theme.ColorNameSeparator)
	d.refit(b.text)
	b.face.Refresh()
	b.text.Refresh()
	// The pill's ends are half its height, and its height has just changed
	// with the text it holds.
	b.face.CornerRadius = b.box.MinSize().Height / 2
	b.box.Refresh()
}

/*
SetHighlight turns the hover on, so the row under the pointer is lit.

It is off unless the program asks for it. A panel is read, not pointed at,
and a list that lit up as a pointer crossed it on the way to somewhere else
would be a list that twitched.
*/
func (d *Drawer) SetHighlight(on bool) {
	if d.highlight == on {
		return
	}
	d.highlight = on
	if !on {
		d.hoverAway()
	}
}

// hoverAt lights the row at a position, and puts out the one that was lit.
func (d *Drawer) hoverAt(at fyne.Position) {
	if !d.highlight || !d.open {
		return
	}
	var now *Row
	for _, r := range d.taprows {
		if r.ID() != "" && rowAt(r, at) {
			now = r
			break
		}
	}
	if now == d.hovered {
		return
	}
	d.hovered.SetHovered(false)
	d.hovered = now
	d.hovered.SetHovered(true)
}

// hoverAway puts out whatever was lit, for a pointer that has left.
func (d *Drawer) hoverAway() {
	if d.hovered == nil {
		return
	}
	d.hovered.SetHovered(false)
	d.hovered = nil
}

/*
SetLeadButton puts a button at the left of the heading, before its words, or
takes it away when the spec is nil.

Where a back belongs. Leaving a page is not one more thing the page offers
alongside what it is for; it is the way out, and the way out is in the same
corner every time.
*/
func (d *Drawer) SetLeadButton(spec *DrawerSpec) {
	if spec == nil {
		if d.lead != nil {
			d.lead.on = nil
			d.lead.box.Hide()
		}
		return
	}
	if d.lead == nil {
		d.lead = newDrawerButton(d)
		d.rebuildTitle()
	}
	apply(d.lead, *spec)
	d.headingChanged()
}

/*
headingChanged measures the heading again now the buttons are drawn.

A button is built hidden and shown when its spec arrives, and the row that
holds it is laid out in between. A container measures what is visible, so
without asking it again the row keeps the width it had when its buttons
were hidden -- nought -- and the buttons are there, correct and invisible.
*/
func (d *Drawer) headingChanged() {
	if d.titled != nil {
		d.titled.Refresh()
	}
	if d.inner != nil {
		d.inner.Refresh()
	}
}

// rebuildTitle puts the buttons beside the heading.
func (d *Drawer) rebuildTitle() {
	row := container.NewHBox()
	for _, b := range d.buttons {
		row.Add(b.box)
	}
	var lead fyne.CanvasObject
	if d.lead != nil {
		lead = d.lead.box
	}
	d.titled = container.New(d.rightPad(), container.NewBorder(
		nil, nil, lead, row, d.title))
	d.inner.Objects[0] = container.NewBorder(d.titled, nil, nil, nil, d.holder)
	d.inner.Refresh()
}

/*
DrawerSpec is one of the drawer's controls: what it reads now, what pressing
it does, and every word it will ever carry.

Words is given so the button can be held at the width of the widest of them.
A control that grows and shrinks as its label changes moves out from under
the pointer that is about to press it again.
*/
type DrawerSpec struct {
	Label string
	On    func()
	Words []string

	// Active marks a toggle that is engaged: the button is filled rather
	// than outlined, so which of its two states it is in can be seen and
	// not only read. A plain button leaves it false.
	Active bool
}

/*
SetButtons draws the row of controls at the right of the drawer's heading,
in the order given: the last is the right-most.

Contextual controls -- a kill that means something only on the page about
one process -- come and go. The row is right-aligned, so one appearing on
the left pushes nothing that was already there out from under the pointer.
*/
func (d *Drawer) SetButtons(specs ...DrawerSpec) {
	for len(d.buttons) < len(specs) {
		d.buttons = append(d.buttons, newDrawerButton(d))
		d.rebuildTitle()
	}
	for i, b := range d.buttons {
		if i >= len(specs) {
			b.on = nil
			b.box.Hide()
			continue
		}
		apply(b, specs[i])
	}
	d.headingChanged()
}

// apply dresses a button in a spec.
func apply(b *drawerButton, spec DrawerSpec) {
	d := b.owner
	b.on = spec.On
	b.widest = 0
	for _, w := range spec.Words {
		t := canvas.NewText(w, nil)
		t.TextSize = b.text.TextSize
		t.TextStyle = b.text.TextStyle
		if x := t.MinSize().Width; x > b.widest {
			b.widest = x
		}
	}
	if b.text.Text != spec.Label {
		b.text.Text = spec.Label
	}
	// Engaged is filled and bold; idle is the quiet ground. It is the fill
	// that changes and not a shade of it, because which of its two states
	// a toggle is in has to be seen at a glance and at ten point.
	b.active = spec.Active
	if spec.Active {
		b.text.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		b.text.TextStyle = fyne.TextStyle{}
	}
	b.box.Show()
	// One place paints a button, so a theme change and a label change
	// cannot leave it looking like two different buttons.
	b.restyle(d)
}
