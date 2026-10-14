package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

var lists = []glance.Choice{
	{ID: "cpu", Label: "CPU", Key: 'c'},
	{ID: "gpu", Label: "GPU", Key: 'g'},
	{ID: "memory", Label: "Mem", Key: 'm'},
}

// switchDrawer is an open drawer holding a switch of three choices, a filter
// and a line, and what the switch reported.
func switchDrawer(t *testing.T) (*glance.Window, *glance.Switch, *glance.Filter, *[]string) {
	t.Helper()
	w, _, _ := drawerWindow(t, glance.Options{Title: "switch"})
	d := w.Panel().Drawer()
	s, f := glance.NewSwitch(), glance.NewFilter()
	s.SetChoices(lists)
	var got []string
	s.OnChanged = func(id string) { got = append(got, id) }
	d.SetTitle("CPU by process")
	d.SetContent(s, f, glance.NewRow("firefox", glance.NoRate()))
	d.SetSwitch(s)
	d.SetFilter(f)
	d.SetOpen(true)
	return w, s, f, &got
}

// Spec 069 R1, R3, R5. A click on a choice selects it and says so once. A
// click on the one selected, on the heading or on a line says nothing, and
// no click in the drawer is a click on a card.
func TestAClickOnAChoiceSelectsIt(t *testing.T) {
	w, s, _, got := switchDrawer(t)
	cards := 0
	w.Panel().OnTapped = func(glance.Tap) { cards++ }
	cv := w.Window().Canvas()
	choices := s.ChoiceObjects()
	require.Equal(t, "cpu", s.Selected(), "the first choice is not the one selected to begin with")

	test.TapCanvas(cv, middle(choices[2]))
	assert.Equal(t, "memory", s.Selected())
	assert.Equal(t, []string{"memory"}, *got)

	test.TapCanvas(cv, middle(choices[2]))
	frame := fyne.CurrentApp().Driver().AbsolutePositionForObject(w.Panel().Drawer().Frame())
	test.TapCanvas(cv, frame.AddXY(6, 6))
	test.TapCanvas(cv, frame.AddXY(6, w.Panel().Drawer().Frame().Size().Height-6))
	assert.Equal(t, []string{"memory"}, *got, "a click that chose nothing was reported")

	test.TapCanvas(cv, middle(choices[1]))
	assert.Equal(t, []string{"memory", "gpu"}, *got)
	assert.Zero(t, cards, "a click in the drawer was reported as a click on a card")
}

// R1. Select is the program's and reports nothing; an ID the switch has not
// is ignored.
func TestSelectingFromTheProgramReportsNothing(t *testing.T) {
	_, s, _, got := switchDrawer(t)
	s.Select("gpu")
	assert.Equal(t, "gpu", s.Selected())
	s.Select("disk")
	assert.Equal(t, "gpu", s.Selected())
	assert.Empty(t, *got)
}

// R3. A choice's key selects it while the drawer is open, and not while it is
// closed or while a filter is being typed, when the key is a character of
// the filter.
func TestAChoicesKeySelectsIt(t *testing.T) {
	w, s, f, got := switchDrawer(t)
	typed := w.Window().Canvas().OnTypedRune()

	typed('g')
	assert.Equal(t, "gpu", s.Selected())
	typed('x')
	assert.Equal(t, []string{"gpu"}, *got)

	typed(glance.DefaultFilterKey)
	typed('m')
	assert.Equal(t, "gpu", s.Selected(), "a key typed into a filter changed the switch")
	assert.Equal(t, "m", f.Text())
	w.Window().Canvas().OnTypedKey()(key(fyne.KeyEscape))

	w.Panel().Drawer().SetOpen(false)
	typed('c')
	assert.Equal(t, "gpu", s.Selected(), "a key changed the switch of a closed drawer")
	assert.Equal(t, []string{"gpu"}, *got)
}

// A switch with no filter beside it has its keys too.
func TestASwitchNeedsNoFilter(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "switch"})
	d := w.Panel().Drawer()
	s := glance.NewSwitch()
	s.SetChoices(lists)
	d.SetContent(s)
	d.SetSwitch(s)
	d.SetOpen(true)
	w.Window().Canvas().OnTypedRune()('m')
	assert.Equal(t, "memory", s.Selected())
}

// R2. Changing the selection moves nothing: every choice is where it was and
// the line is the size it was, whichever is selected.
func TestChangingTheSelectionMovesNothing(t *testing.T) {
	_, s, _, _ := switchDrawer(t)
	type place struct {
		at   fyne.Position
		size fyne.Size
	}
	places := func() []place {
		var out []place
		for _, o := range s.ChoiceObjects() {
			out = append(out, place{o.Position(), o.Size()})
		}
		return out
	}
	before, line := places(), s.Object().MinSize()
	for _, c := range lists {
		s.Select(c.ID)
		assert.Equal(t, before, places(), "selecting %s moved a choice", c.ID)
		assert.Equal(t, line, s.Object().MinSize(), "selecting %s resized the line", c.ID)
	}
	for i := 1; i < len(before); i++ {
		assert.Greater(t, before[i].at.X, before[i-1].at.X+before[i-1].size.Width,
			"choice %d is not clear of the one before it", i)
	}
}

// R4. A switch with one choice or none is not drawn and takes no height in a
// drawer; with two it does.
func TestASwitchWithNothingToChooseTakesNoRoom(t *testing.T) {
	fynetest.App(t)
	line := glance.NewRow("firefox", glance.NoRate())
	height := func(n int) float32 {
		s := glance.NewSwitch()
		s.SetChoices(lists[:n])
		w, _, _ := drawerWindow(t, glance.Options{Title: "switch"})
		d := w.Panel().Drawer()
		d.SetContent(s, line)
		d.SetOpen(true)
		assert.Equal(t, n >= 2, s.Object().Visible(), "%d choices", n)
		return float32(d.Fit(line, s))
	}
	none, one, three := height(0), height(1), height(3)
	assert.Equal(t, none, one)
	assert.Less(t, three, none, "a switch of three choices took no line of the drawer's room")
}

// The selection outlives a change of choices that still has it, and goes to
// the first when it has gone. Two choices with one ID are a mistake.
func TestTheSelectionFollowsTheChoices(t *testing.T) {
	fynetest.App(t)
	s := glance.NewSwitch()
	s.SetChoices(lists)
	s.Select("memory")
	s.SetChoices([]glance.Choice{lists[0], lists[2]})
	assert.Equal(t, "memory", s.Selected())
	s.SetChoices(lists[:2])
	assert.Equal(t, "cpu", s.Selected())
	assert.Panics(t, func() { s.SetChoices([]glance.Choice{lists[0], lists[0]}) })
}
