package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// Choice is one of the things a Switch chooses between.
type Choice struct {
	// ID is what the program knows the choice by.
	ID string
	// Label is what is drawn: a word, "CPU".
	Label string
	// Key selects the choice from the keyboard while the drawer is open.
	// Zero for a choice with no key.
	Key rune
}

/*
Switch is one line in a drawer that chooses what the drawer shows: a few named
choices, one of them selected (spec 069). A card with several details has one
drawer, and this is how a person moves between them without going back to the
card.

**A choice is clicked, or its key is typed.** The panel's catcher takes every
click, so nothing in a drawer is clicked directly; Drawer.SetSwitch points the
panel's clicks and the window's keys here, as SetFilter does for a filter.

**Changing the selection moves nothing.** Each choice takes the width of its
label in bold whether it is selected or not.

A switch with fewer than two choices is not drawn and takes no height: there
is nothing to choose.

The switch does not hold what the drawer shows. OnChanged says a person chose
and what the drawer does about it is the program's.
*/
type Switch struct {
	themed

	// OnChanged is called, on the UI thread, when a person selects a choice
	// that was not selected. Select does not call it.
	OnChanged func(id string)

	choices  []Choice
	selected string
	texts    []*canvas.Text
	box      *fyne.Container
}

// NewSwitch builds a switch with nothing to choose.
func NewSwitch() *Switch {
	s := &Switch{}
	s.box = container.New(&switchLayout{sw: s})
	s.box.Hide()
	return s
}

// Object is the switch's line, for the drawer's content.
func (s *Switch) Object() fyne.CanvasObject { return s.box }

// SetTheme gives the switch a face of its own and repaints in it.
func (s *Switch) SetTheme(th fyne.Theme) {
	s.themed.SetTheme(th)
	s.Restyle()
}

/*
SetChoices names what the switch chooses between, in the order drawn. The
selection is kept when the choice selected is still among them and is the
first otherwise. Two choices with one ID are a programming error and panic.
*/
func (s *Switch) SetChoices(choices []Choice) {
	seen := make(map[string]bool, len(choices))
	for _, c := range choices {
		if seen[c.ID] {
			panic("glance: two choices of a switch have the ID " + c.ID)
		}
		seen[c.ID] = true
	}
	s.choices = append(s.choices[:0:0], choices...)
	if !seen[s.selected] {
		s.selected = ""
		if len(choices) > 0 {
			s.selected = choices[0].ID
		}
	}
	objects := make([]fyne.CanvasObject, len(choices))
	s.texts = make([]*canvas.Text, len(choices))
	for i, c := range choices {
		s.texts[i] = canvas.NewText(c.Label, nil)
		objects[i] = s.texts[i]
	}
	s.box.Objects = objects
	if len(choices) < 2 {
		s.box.Hide()
	} else {
		s.box.Show()
	}
	s.Restyle()
}

// Choices are what the switch chooses between.
func (s *Switch) Choices() []Choice { return append([]Choice(nil), s.choices...) }

// Selected is the ID of the choice selected, empty for a switch with none.
func (s *Switch) Selected() string { return s.selected }

// Select selects a choice for the program. It does not call OnChanged, and an
// ID the switch does not have is ignored.
func (s *Switch) Select(id string) {
	if id == s.selected || !s.has(id) {
		return
	}
	s.selected = id
	s.Restyle()
}

func (s *Switch) has(id string) bool {
	for _, c := range s.choices {
		if c.ID == id {
			return true
		}
	}
	return false
}

// choose is a person selecting a choice: a click or a key.
func (s *Switch) choose(id string) {
	if id == s.selected {
		return
	}
	s.Select(id)
	if s.OnChanged != nil {
		s.OnChanged(id)
	}
}

// TypedRune takes a character: a choice's key selects it. It reports whether
// the character was a choice's.
func (s *Switch) TypedRune(r rune) bool {
	if len(s.choices) < 2 || r == 0 {
		return false
	}
	for _, c := range s.choices {
		if c.Key == r {
			s.choose(c.ID)
			return true
		}
	}
	return false
}

// tappedAt selects the choice at a position on the canvas, if one is there.
func (s *Switch) tappedAt(at fyne.Position) {
	if !s.box.Visible() {
		return
	}
	drv := fyne.CurrentApp().Driver()
	for i, t := range s.texts {
		pos, size := drv.AbsolutePositionForObject(t), t.Size()
		if at.X >= pos.X && at.X < pos.X+size.Width && at.Y >= pos.Y && at.Y < pos.Y+size.Height {
			s.choose(s.choices[i].ID)
			return
		}
	}
}

// Restyle repaints the choices in the current theme and selection.
func (s *Switch) Restyle() {
	for i, t := range s.texts {
		t.TextSize = s.textSize()
		t.TextStyle = fyne.TextStyle{}
		t.Color = s.colour(theme.ColorNamePlaceHolder)
		if s.choices[i].ID == s.selected {
			t.TextStyle = s.bold(fyne.TextStyle{})
			t.Color = s.colour(theme.ColorNameForeground)
		}
		t.FontSource = s.Theme().Font(t.TextStyle)
		t.Refresh()
	}
	s.box.Refresh()
}

// slot is the room a choice takes: its label in bold, selected or not.
func (s *Switch) slot(t *canvas.Text) fyne.Size {
	style := s.bold(fyne.TextStyle{})
	size, _ := fyne.CurrentApp().Driver().RenderedTextSize(t.Text, s.textSize(), style, s.Theme().Font(style))
	return size
}

// switchLayout puts the choices in a row, each in its slot, a text size
// apart.
type switchLayout struct{ sw *Switch }

func (l *switchLayout) gap() float32 { return l.sw.textSize() }

func (l *switchLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	var out fyne.Size
	for i, t := range l.sw.texts {
		slot := l.sw.slot(t)
		if i > 0 {
			out.Width += l.gap()
		}
		out.Width += slot.Width
		out.Height = max(out.Height, slot.Height)
	}
	return out
}

func (l *switchLayout) Layout(_ []fyne.CanvasObject, _ fyne.Size) {
	x := float32(0)
	for _, t := range l.sw.texts {
		slot := l.sw.slot(t)
		t.Move(fyne.NewPos(x, 0))
		t.Resize(slot)
		x += slot.Width + l.gap()
	}
}
