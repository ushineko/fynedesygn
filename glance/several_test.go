package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// Spec 079: several drawers beside the cards.

// x is where an object starts across the window.
func x(o fyne.CanvasObject) float32 { return o.Position().X }

// R1. The first drawer is there from the start; AddDrawer makes more, closed,
// and Drawers lists them, the first first.
func TestAddDrawerMakesAnotherClosedDrawer(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	require.Len(t, p.Drawers(), 1)
	second := p.AddDrawer()
	assert.False(t, second.Open())
	assert.Equal(t, []*glance.Drawer{p.Drawer(), second}, p.Drawers())
	assert.Nil(t, p.Outermost())
	assert.Zero(t, p.Opened())
}

// R2, R3. Two drawers on the right: the window is the cards and two Reaches,
// the first opened next to the cards and the second beyond it, and the cards
// stay where they were.
func TestTwoDrawersStandInARowOnTheRight(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	first, second := p.Drawer(), p.AddDrawer()
	first.SetSide(glance.Right)
	closed := w.Window().Canvas().Size()
	cards := p.CardsWidth()

	second.SetOpen(true)
	first.SetOpen(true)
	size := w.Window().Canvas().Size()
	assert.InDelta(t, closed.Width+2*first.Reach(), size.Width, 0.01)
	assert.Equal(t, closed.Height, size.Height)
	assert.Zero(t, p.CardsOrigin().X)
	assert.InDelta(t, cards, p.CardsWidth(), 0.01, "the cards changed width")

	// Opened first, second is next to the cards.
	assert.InDelta(t, cards+first.Reach()-second.Frame().Size().Width, x(second.Frame()), 0.01)
	assert.InDelta(t, x(second.Frame())+first.Reach(), x(first.Frame()), 0.01)
	assert.InDelta(t, size.Width, x(first.Frame())+first.Frame().Size().Width, 0.01)
	assert.Same(t, first, p.Outermost())
}

// R2, R4. On the left the row grows away from the cards, which stand off the
// window's edge by both drawers.
func TestTwoDrawersStandInARowOnTheLeft(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	p.SetSelfMoving(true)
	first, second := p.Drawer(), p.AddDrawer()
	first.SetSide(glance.Left)

	first.SetOpen(true)
	assert.InDelta(t, first.Reach(), p.CardsOrigin().X, 0.01)
	second.SetOpen(true)
	assert.InDelta(t, 2*first.Reach(), p.CardsOrigin().X, 0.01)
	assert.InDelta(t, 0, x(second.Frame()), 0.01, "the drawer opened last is not at the window's edge")
	assert.InDelta(t, first.Reach(), x(first.Frame()), 0.01)
}

// R2, R4. Closing the inner drawer moves the outer one in, and takes one
// Reach off the window.
func TestClosingTheInnerDrawerMovesTheOuterIn(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	first, second := p.Drawer(), p.AddDrawer()
	first.SetSide(glance.Right)
	first.SetOpen(true)
	inner := x(first.Frame())
	second.SetOpen(true)
	two := w.Window().Canvas().Size().Width

	first.SetOpen(false)
	assert.InDelta(t, two-first.Reach(), w.Window().Canvas().Size().Width, 0.01)
	assert.InDelta(t, inner, x(second.Frame()), 0.01, "the outer drawer did not move in")
	assert.Same(t, second, p.Outermost())
	assert.Equal(t, 1, p.Opened())
}

// R5. The side is the panel's: naming it on one drawer names it for all.
func TestTheSideIsThePanels(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	p.SetSelfMoving(true)
	first, second := p.Drawer(), p.AddDrawer()
	second.SetSide(glance.Left)
	assert.Equal(t, glance.Left, first.Side())
	assert.False(t, first.Auto())

	first.SetOpen(true)
	second.SetOpen(true)
	assert.InDelta(t, 2*first.Reach(), p.CardsOrigin().X, 0.01, "the drawers are not both on the left")
}

// R5. A row that is out keeps its side when the next drawer is placed by the
// room.
func TestARowThatIsOutKeepsItsSide(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	p.SetSelfMoving(true)
	first, second := p.Drawer(), p.AddDrawer()
	assert.Equal(t, glance.Right, first.Choose(0, 1000))
	first.SetOpen(true)
	assert.Equal(t, glance.Right, second.Choose(1000, 0), "the second drawer crossed the cards")
}

// R6. With none open a drawer always fits; with one open, one more fits where
// its side has a Reach to spare.
func TestFitsIsTheRoomOnTheRowsSide(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	p.SetSelfMoving(true)
	d := p.Drawer()
	reach := d.Reach()
	assert.True(t, p.Fits(0, 0), "the first drawer must open whatever the room")

	d.SetSide(glance.Right)
	d.SetOpen(true)
	assert.True(t, p.Fits(0, reach))
	assert.False(t, p.Fits(1000, reach-1), "room on the other side was counted")
	d.SetOpen(false)

	d.SetSide(glance.Left)
	d.SetOpen(true)
	assert.True(t, p.Fits(reach, 0))
	assert.False(t, p.Fits(reach-1, 1000))
}

// R9. Outermost is the last opened, and after it closes the one before.
func TestOutermostIsTheLastOpened(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	a, b, c := p.Drawer(), p.AddDrawer(), p.AddDrawer()
	b.SetOpen(true)
	a.SetOpen(true)
	c.SetOpen(true)
	assert.Same(t, c, p.Outermost())
	c.SetOpen(false)
	assert.Same(t, a, p.Outermost())
}

// R10. A click with Shift held says so; one without does not.
func TestAClickSaysWhetherShiftWasHeld(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "several"})
	var got []glance.Tap
	w.Panel().OnTapped = func(tap glance.Tap) { got = append(got, tap) }
	cv := w.Window().Canvas()

	test.TapCanvas(cv, middle(cards[0].Object()))
	w.Panel().SetShift(true)
	test.TapCanvas(cv, middle(cards[1].Object()))

	require.Len(t, got, 2)
	assert.False(t, got[0].Shift)
	assert.True(t, got[1].Shift)
	assert.Same(t, cards[1], got[1].Card)
}

// R11. A Shift-click on a choice is reported to OnShifted, the selected one
// included, and changes nothing; without OnShifted it is a click.
func TestAShiftClickOnAChoiceIsReportedAndSelectsNothing(t *testing.T) {
	w, s, _, changed := switchDrawer(t)
	var shifted []string
	s.OnShifted = func(id string) { shifted = append(shifted, id) }
	cv := w.Window().Canvas()
	choices := s.ChoiceObjects()
	w.Panel().SetShift(true)

	test.TapCanvas(cv, middle(choices[1]))
	test.TapCanvas(cv, middle(choices[0]))
	assert.Equal(t, []string{"gpu", "cpu"}, shifted)
	assert.Empty(t, *changed)
	assert.Equal(t, "cpu", s.Selected())

	s.OnShifted = nil
	test.TapCanvas(cv, middle(choices[2]))
	assert.Equal(t, "memory", s.Selected())
	assert.Equal(t, []string{"memory"}, *changed)
}

// R12. A click in the second drawer is that drawer's: its switch, and not the
// first's, and not a card's.
func TestAClickInTheSecondDrawerIsItsOwn(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	cards := 0
	p.OnTapped = func(glance.Tap) { cards++ }
	first, second := p.Drawer(), p.AddDrawer()
	s1, s2 := glance.NewSwitch(), glance.NewSwitch()
	var got1, got2 []string
	for _, sw := range []struct {
		d   *glance.Drawer
		s   *glance.Switch
		got *[]string
	}{{first, s1, &got1}, {second, s2, &got2}} {
		sw.s.SetChoices(lists)
		got := sw.got
		sw.s.OnChanged = func(id string) { *got = append(*got, id) }
		sw.d.SetContent(sw.s)
		sw.d.SetSwitch(sw.s)
		sw.d.SetOpen(true)
	}

	test.TapCanvas(w.Window().Canvas(), middle(s2.ChoiceObjects()[1]))
	assert.Empty(t, got1)
	assert.Equal(t, []string{"gpu"}, got2)
	assert.Zero(t, cards)
}

// R13. The keys go to the drawer opened last, then to the one clicked, and
// when the keyed one closes, to the one opened last of those left.
func TestTheKeysGoToOneDrawer(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	first, second := p.Drawer(), p.AddDrawer()
	f1, f2 := glance.NewFilter(), glance.NewFilter()
	first.SetContent(f1)
	first.SetFilter(f1)
	second.SetContent(f2)
	second.SetFilter(f2)
	first.SetOpen(true)
	second.SetOpen(true)
	typed, key := w.Window().Canvas().OnTypedRune(), w.Window().Canvas().OnTypedKey()
	escape := &fyne.KeyEvent{Name: fyne.KeyEscape}

	typed(glance.DefaultFilterKey)
	typed('a')
	assert.Equal(t, "a", f2.Text())
	assert.Empty(t, f1.Text())
	key(&fyne.KeyEvent{Name: fyne.KeyReturn})

	frame := fyne.CurrentApp().Driver().AbsolutePositionForObject(first.Frame())
	test.TapCanvas(w.Window().Canvas(), frame.AddXY(6, first.Frame().Size().Height-6))
	typed(glance.DefaultFilterKey)
	typed('b')
	assert.Equal(t, "b", f1.Text(), "a click in the first drawer did not give it the keys")
	key(escape)

	first.SetOpen(false)
	typed(glance.DefaultFilterKey)
	typed('c')
	assert.Equal(t, "ac", f2.Text(), "closing the keyed drawer did not hand the keys on")
}

// R15. Closing one of two drawers keeps the keyboard; closing the last gives
// it back.
func TestOnlyTheLastDrawerClosingGivesTheKeyboardBack(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "several"})
	p := w.Panel()
	first, second := p.Drawer(), p.AddDrawer()
	yields := 0
	first.SetYield(func() { yields++ })
	second.SetYield(func() { yields++ })
	first.SetOpen(true)
	second.SetOpen(true)

	first.SetOpen(false)
	assert.Zero(t, yields)
	second.SetOpen(false)
	assert.Equal(t, 1, yields)
}
