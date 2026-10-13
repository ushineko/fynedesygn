package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// drawerWindow is a shown window of two drawn cards, each of two rows, and a
// meter in the second.
func drawerWindow(t *testing.T, o glance.Options) (*glance.Window, []*glance.Card, *glance.Meter) {
	t.Helper()
	w := newTestWindow(t, o)
	// A glance window is a splash window and has no padding; the test
	// driver's ordinary one has, and would make the window's content smaller
	// than the size the panel asked for.
	w.Window().SetPadded(false)
	net := glance.NewCard("Bandwidth")
	net.AddRow(named("down", "eno2 down"), named("up", "eno2 up"))
	cooler := glance.NewCard("Cooler")
	cooler.AddRow(named("cpu", "CPU"), named("gpu", "GPU"))
	meter := glance.NewMeter("5h", 40)
	cooler.Add(meter)
	w.Panel().Add(net, cooler)
	net.SetAvailable(true)
	cooler.SetAvailable(true)
	w.Window().Show()
	w.Panel().Resize()
	return w, []*glance.Card{net, cooler}, meter
}

// middle is the middle of an object, on the window's canvas.
func middle(o fyne.CanvasObject) fyne.Position {
	at := fyne.CurrentApp().Driver().AbsolutePositionForObject(o)
	return at.AddXY(o.Size().Width/2, o.Size().Height/2)
}

// Spec 068 R1, R4. Opening the drawer widens the window and leaves its
// height alone; closing puts back the size it had.
func TestADrawerWidensTheWindowAndClosingPutsItBack(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	closed := w.Window().Canvas().Size()
	least := w.Panel().Content().MinSize().Width

	w.Panel().Drawer().SetOpen(true)
	open := w.Window().Canvas().Size()
	assert.Greater(t, open.Width, closed.Width+glance.MinCardWidth(), "the window did not grow by a drawer")
	assert.Equal(t, closed.Height, open.Height, "the drawer changed the window's height")
	assert.Equal(t, open, w.Panel().Size())
	// What the content says it needs is what Fyne fits a window to, so it has
	// to say the same.
	// The floor is the panel's, so the two widths differ; what the drawer
	// adds to each is the same.
	assert.InDelta(t, open.Width-closed.Width, w.Panel().Content().MinSize().Width-least, 0.01)

	w.Panel().Drawer().SetOpen(false)
	assert.Equal(t, closed, w.Window().Canvas().Size(), "closing did not put the window back")
}

// R2. The drawer is the cards' height, with one card and with two, and a card
// that goes shortens both.
func TestADrawerIsTheHeightOfTheCards(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d := w.Panel().Drawer()
	d.SetOpen(true)

	two := d.Frame().Size().Height
	assert.InDelta(t, w.Catcher().Size().Height, two, 0.01)
	assert.InDelta(t, cards[1].Object().Position().Y+cards[1].Object().Size().Height, two, 0.01,
		"the drawer does not end where the last card does")

	cards[1].SetAllowed(false)
	one := d.Frame().Size().Height
	assert.Less(t, one, two, "a card went and the drawer kept its height")
	assert.InDelta(t, cards[0].Object().Size().Height, one, 0.01)
}

// R2. The drawer never sets the height: content taller than the cards leaves
// the window as tall as the cards.
func TestADrawersContentDoesNotSetTheHeight(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	closed := w.Window().Canvas().Size()

	var pieces []glance.Piece
	for range 60 {
		pieces = append(pieces, glance.NewRow("process", glance.NoRate()))
	}
	d := w.Panel().Drawer()
	d.SetContent(pieces...)
	d.SetOpen(true)

	assert.Equal(t, closed.Height, w.Window().Canvas().Size().Height)
	assert.InDelta(t, closed.Height, w.Panel().Content().MinSize().Height, 0.01,
		"the content's own minimum is the drawer's, and Fyne grows a window to that")
}

// R2. Fit is the number of lines that go in the room there is: that many do
// and one more does not.
func TestFitIsTheLinesThatGoInTheRoom(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d := w.Panel().Drawer()
	line, filter := glance.NewRow("process", glance.NoRate()), glance.NewFilter()

	pad := fyne.CurrentApp().Settings().Theme().Size("padding")
	each := line.Object().MinSize().Height + pad
	room := d.Available().Height

	n := d.Fit(line)
	require.Positive(t, n)
	assert.LessOrEqual(t, float32(n)*each-pad, room)
	assert.Greater(t, float32(n+1)*each-pad, room)

	assert.Less(t, d.Fit(line, filter), n+1, "a piece above the list took no room")
	assert.LessOrEqual(t, d.Fit(line, filter), n)
}

// R4, R9. On the right the cards stay at the window's left edge; on the left
// they stand off it by the drawer and the gap.
func TestTheCardsStandWhereTheDrawerIsNot(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	p, d := w.Panel(), w.Panel().Drawer()
	closed := p.Size().Width

	d.SetSide(glance.Right)
	d.SetOpen(true)
	assert.Zero(t, p.CardsOrigin().X)
	assert.InDelta(t, p.Size().Width-d.Frame().Size().Width, d.Frame().Position().X, 0.01)
	d.SetOpen(false)

	p.SetSelfMoving(true)
	d.SetSide(glance.Left)
	d.SetOpen(true)
	assert.InDelta(t, p.Size().Width-closed, p.CardsOrigin().X, 0.01)
	assert.Zero(t, d.Frame().Position().X)
}

// R5. A width the user dragged is a width of the cards: it is there after
// the drawer has opened and closed, and the open window is that much wider.
func TestADraggedWidthSurvivesTheDrawer(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer", Resizable: true})
	d := w.Panel().Drawer()

	wide := w.Panel().Size().Width + 120
	w.Window().Resize(fyne.NewSize(wide, w.Panel().Size().Height))
	w.Panel().Resize()
	require.InDelta(t, wide, w.Window().Canvas().Size().Width, 0.01)

	d.SetOpen(true)
	assert.Greater(t, w.Window().Canvas().Size().Width, wide+glance.MinCardWidth())
	w.Panel().Resize() // a reading arrives while it is open

	d.SetOpen(false)
	w.Panel().Resize()
	assert.InDelta(t, wide, w.Window().Canvas().Size().Width, 0.01,
		"the user's width was lost to the drawer")
}

// A card whose height depends on its width keeps its height when the drawer
// opens and closes. Four cells are two to a row at the card's width and one
// to a row at a few units; the cards were laid out at a few units on the way
// to opening, in a window not yet widened, and the window was sized to four
// rows of cells.
func TestOpeningTheDrawerDoesNotMeasureTheCardsAtTheWrongWidth(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "drawer"})
	w.Window().SetPadded(false)
	devices := glance.NewCard("Peripherals")
	grid := glance.NewCellGrid()
	for _, name := range []string{"Mouse", "Keyboard", "Headset", "Controller"} {
		grid.Add(glance.NewCell(name, "100 %"))
	}
	devices.Add(grid)
	w.Panel().Add(devices)
	devices.SetAvailable(true)
	w.Window().Show()
	w.Panel().Resize()
	closed := w.Window().Canvas().Size()
	d := w.Panel().Drawer()

	for _, side := range []glance.Side{glance.Right, glance.Left} {
		w.Panel().SetSelfMoving(side == glance.Left)
		d.SetSide(side)
		d.SetOpen(true)
		open := w.Window().Canvas().Size()
		assert.Greater(t, open.Width, closed.Width)
		assert.InDelta(t, closed.Height, open.Height, 0.01,
			"the window changed height when the drawer opened on side %d", side)

		d.SetOpen(false)
		back := w.Window().Canvas().Size()
		assert.InDelta(t, closed.Width, back.Width, 0.01)
		assert.InDelta(t, closed.Height, back.Height, 0.01,
			"the window changed height when the drawer closed on side %d", side)
	}
}

// A window put back to the width it had with the drawer open is not a window
// somebody widened. On Wayland the compositor answers a change of focus with
// the size it last knew; closing a drawer and giving the keyboard back at
// once crossed the two, the window went back to its open width, and the
// cards took that width as the user's and kept it, growing by a drawer at
// every opening after. The panel asks for its own width again.
func TestAWindowPutBackToItsOpenWidthIsNotAChosenWidth(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer", Resizable: true})
	p, d := w.Panel(), w.Panel().Drawer()
	closed := w.Window().Canvas().Size()

	for range 3 {
		d.SetOpen(true)
		p.Resize() // a reading arrives while it is open
		open := w.Window().Canvas().Size()
		require.Greater(t, open.Width, closed.Width)

		d.SetOpen(false)
		w.Window().Resize(open) // the compositor's answer, with the old size
		p.Resize()              // the next reading
		assert.InDelta(t, closed.Width, w.Window().Canvas().Size().Width, 0.01,
			"the open width was kept as the cards' width")
	}

	// And the other way: put back to the closed width just after opening.
	d.SetOpen(true)
	open := w.Window().Canvas().Size()
	w.Window().Resize(closed)
	p.Resize()
	assert.InDelta(t, open.Width, w.Window().Canvas().Size().Width, 0.01)
	d.SetOpen(false)

	// A width somebody did choose is still kept.
	w.Window().Resize(fyne.NewSize(closed.Width+77, closed.Height))
	p.Resize()
	assert.InDelta(t, closed.Width+77, w.Window().Canvas().Size().Width, 0.01)
}

// A window is a whole number of pixels wide, so the width it has is rarely
// the width asked for to the last fraction. That fraction is nobody's choice
// of width: it is not kept, and a drawer opened and closed any number of
// times leaves the cards the width they were.
func TestAFractionOfAPixelIsNotAWidthSomebodyChose(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer", Resizable: true})
	p, d := w.Panel(), w.Panel().Drawer()
	cards := p.Size().Width
	height := p.Size().Height

	// The system rounds the window to its pixels, as it does at a scale that
	// is not a whole number.
	rounded := func() {
		s := w.Window().Canvas().Size()
		w.Window().Resize(fyne.NewSize(s.Width+0.4, s.Height))
	}
	rounded()
	p.Resize() // a reading arrives
	require.InDelta(t, cards, w.Window().Canvas().Size().Width, 0.01, "the rounding was kept as a width")

	for range 3 {
		d.SetOpen(true)
		rounded()
		p.Resize()
		d.SetOpen(false)
		rounded()
		p.Resize()
	}
	assert.InDelta(t, cards, w.Window().Canvas().Size().Width, 0.01,
		"the cards grew with each opening of the drawer")
	assert.InDelta(t, height, w.Window().Canvas().Size().Height, 0.01)

	// A whole pixel and more is still a width somebody chose.
	w.Window().Resize(fyne.NewSize(cards+40, height))
	p.Resize()
	assert.InDelta(t, cards+40, w.Window().Canvas().Size().Width, 0.01)
}

// R8. A drawer that chooses by room takes the right if it fits there, the
// left if it fits only there, and the roomier side if it fits on neither.
func TestADrawerChoosesTheSideWithRoom(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	p, d := w.Panel(), w.Panel().Drawer()
	p.SetSelfMoving(true)

	closed := p.Size().Width
	d.SetOpen(true)
	need := p.Size().Width - closed
	d.SetOpen(false)
	require.Positive(t, need)

	for _, c := range []struct {
		name        string
		left, right float32
		want        glance.Side
	}{
		{"room on both", 2 * need, 2 * need, glance.Right},
		{"room on the right only", 0, need, glance.Right},
		{"room on the left only", need, need - 1, glance.Left},
		{"on neither, more on the left", need - 1, need / 2, glance.Left},
		{"on neither, more on the right", need / 2, need - 1, glance.Right},
		{"on neither, the same", 10, 10, glance.Right},
	} {
		assert.Equal(t, c.want, d.Choose(c.left, c.right), c.name)
		assert.True(t, d.Auto(), c.name)
	}
}

// R8. A side asked for by name is kept, whatever the room; SetAuto gives the
// choice back.
func TestANamedSideIsKept(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d := w.Panel().Drawer()
	w.Panel().SetSelfMoving(true)

	d.SetSide(glance.Left)
	assert.False(t, d.Auto())
	assert.Equal(t, glance.Left, d.Choose(0, 5000))

	d.SetAuto()
	assert.Equal(t, glance.Right, d.Choose(0, 5000))
}

// R9. Where the window cannot move itself and nothing holds its right edge,
// a left drawer opens on the right: the other choice is cards that jump.
func TestADrawerThatCannotGrowLeftwardsOpensOnTheRight(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	p, d := w.Panel(), w.Panel().Drawer()
	p.SetSelfMoving(false)

	d.SetSide(glance.Left)
	d.SetOpen(true)
	assert.Equal(t, glance.Right, d.Side())
	assert.Zero(t, p.CardsOrigin().X)
	d.SetOpen(false)

	d.SetEdgeHeld(true)
	d.SetSide(glance.Left)
	d.SetOpen(true)
	assert.Equal(t, glance.Left, d.Side())
	assert.Positive(t, p.CardsOrigin().X)
}

// R3, R6. The drawer's width is a factor of the panel's own text size.
func TestADrawerIsWiderInALargerFace(t *testing.T) {
	width := func(size float32) float32 {
		w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
		w.Panel().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: size}))
		w.Panel().Drawer().SetOpen(true)
		return w.Panel().Drawer().Frame().Size().Width
	}
	small, large := width(10), width(16)
	assert.InDelta(t, 10*glance.DrawerWidthInEms, small, 0.5)
	assert.InDelta(t, 16*glance.DrawerWidthInEms, large, 0.5)
}

// R6. What the drawer holds takes the panel's theme, when it is set before
// the content and when it is set after.
func TestADrawersContentTakesThePanelsTheme(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d := w.Panel().Drawer()

	before := glance.NewRow("before", glance.NoRate())
	d.SetContent(before)
	small := before.Object().MinSize().Height

	w.Panel().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 20}))
	assert.Greater(t, before.Object().MinSize().Height, small, "content set before the theme kept the old face")

	after := glance.NewRow("after", glance.NoRate())
	d.SetContent(after)
	assert.InDelta(t, before.Object().MinSize().Height, after.Object().MinSize().Height, 0.01,
		"content set after the theme did not take it")
}

// The drawer has a heading, as a card has.
func TestADrawerHasAHeading(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d := w.Panel().Drawer()
	d.SetTitle("CPU by process")
	d.SetOpen(true)

	assert.Equal(t, "CPU by process", d.Title())
	assert.Contains(t, texts(d.Frame()), "CPU by process")
}

// R11. A click is reported with its card, and with the row or the meter under
// it; a click on the heading has neither.
func TestAClickIsReportedWithWhatItLandedOn(t *testing.T) {
	w, cards, meter := drawerWindow(t, glance.Options{Title: "drawer"})
	var got []glance.Tap
	w.Panel().OnTapped = func(tap glance.Tap) { got = append(got, tap) }
	cv := w.Window().Canvas()

	test.TapCanvas(cv, middle(cards[1].RowByID("gpu").Object()))
	test.TapCanvas(cv, middle(meter.Object()))
	test.TapCanvas(cv, fyne.CurrentApp().Driver().AbsolutePositionForObject(cards[0].Object()).AddXY(4, 4))

	require.Len(t, got, 3)
	assert.Same(t, cards[1], got[0].Card)
	assert.Same(t, cards[1].RowByID("gpu"), got[0].Row)
	assert.Nil(t, got[0].Meter)

	assert.Same(t, cards[1], got[1].Card)
	assert.Nil(t, got[1].Row)
	assert.Same(t, meter, got[1].Meter)

	assert.Same(t, cards[0], got[2].Card)
	assert.Nil(t, got[2].Row)
	assert.Nil(t, got[2].Meter)
}

// R11. A click where a hidden card was, or on the drawer, is not a click on
// a card.
func TestAClickOffTheCardsIsNotReported(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	n := 0
	w.Panel().OnTapped = func(glance.Tap) { n++ }
	cv := w.Window().Canvas()

	second := middle(cards[1].RowByID("gpu").Object())
	cards[1].SetAllowed(false)
	test.TapCanvas(cv, second)

	d := w.Panel().Drawer()
	d.SetOpen(true)
	test.TapCanvas(cv, middle(d.Frame()))

	assert.Zero(t, n)
}

// R11. A window with no menu reports clicks too, and one whose program wants
// none does nothing with them.
func TestAClickNeedsNoMenuAndNoListener(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	at := middle(cards[0].RowByID("down").Object())

	assert.NotPanics(t, func() { test.TapCanvas(w.Window().Canvas(), at) })

	n := 0
	w.Panel().OnTapped = func(glance.Tap) { n++ }
	test.TapCanvas(w.Window().Canvas(), at)
	assert.Equal(t, 1, n)
}

// R12, R13. The catcher is what a drag lands on, so a press that moves is a
// drag to the toolkit and not a tap.
func TestTheCatcherTakesDragsSoThatADragIsNotATap(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})

	_, drags := w.Catcher().(fyne.Draggable)
	_, taps := w.Catcher().(fyne.Tappable)
	assert.True(t, drags, "a press that moves would be reported as a click")
	assert.True(t, taps)
}

/*
R14. After the compositor has moved the window under a pointer that has not
moved, the next click arrives at the place the pointer had before (quirk 48).
A second click on the row that opened a left drawer is still a click on that
row.

The drawer here is the program's: open on a click, close on a second click of
the same row.
*/
func TestASecondClickWithoutMovingThePointerIsOnTheSameRow(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	p, d := w.Panel(), w.Panel().Drawer()
	p.SetSelfMoving(false)
	d.SetEdgeHeld(true)
	d.SetSide(glance.Left)

	row := cards[0].RowByID("up")
	var got []*glance.Row
	p.OnTapped = func(tap glance.Tap) {
		got = append(got, tap.Row)
		d.SetOpen(!d.Open())
	}
	cv := w.Window().Canvas()
	at := middle(row.Object())

	test.TapCanvas(cv, at) // opens: the cards move right under the pointer
	require.True(t, d.Open())
	require.Greater(t, middle(row.Object()).X, at.X+glance.MinCardWidth())

	test.TapCanvas(cv, at) // the pointer has not moved
	require.Len(t, got, 2)
	assert.Same(t, row, got[1], "the second click was read where the pointer used to be")
	assert.False(t, d.Open())

	test.TapCanvas(cv, at) // nor now, and the cards are back where they were
	require.Len(t, got, 3)
	assert.Same(t, row, got[2], "a click after the window came back was moved off its row")
}

// R14. A click somewhere else is a pointer that moved, and is read where it
// says it is.
func TestAClickElsewhereIsReadWhereItIs(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	p, d := w.Panel(), w.Panel().Drawer()
	p.SetSelfMoving(false)
	d.SetEdgeHeld(true)
	d.SetSide(glance.Left)

	var got []glance.Tap
	p.OnTapped = func(tap glance.Tap) {
		got = append(got, tap)
		d.SetOpen(true)
	}
	cv := w.Window().Canvas()

	test.TapCanvas(cv, middle(cards[0].RowByID("up").Object()))
	test.TapCanvas(cv, middle(cards[1].RowByID("cpu").Object()))

	require.Len(t, got, 2)
	assert.Same(t, cards[1].RowByID("cpu"), got[1].Row)
}

// R20. Closing the drawer gives the keyboard back, through the program's way
// where the window has none of its own; opening does not.
func TestClosingTheDrawerGivesTheKeyboardBack(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d := w.Panel().Drawer()
	yields := 0
	d.SetYield(func() { yields++ })

	d.SetOpen(true)
	assert.Zero(t, yields)
	d.SetOpen(false)
	assert.Equal(t, 1, yields)
	d.SetOpen(false)
	assert.Equal(t, 1, yields, "a drawer that was already closed yielded again")
}

// A panel that is not in a window has a drawer that opens and measures, and
// has nothing to resize.
func TestAPanelWithNoWindowHasADrawerThatMeasures(t *testing.T) {
	test.NewTempApp(t)
	p := glance.NewPanel(0)
	c := glance.NewCard("Bandwidth")
	c.AddRow(named("down", "eno2 down"))
	p.Add(c)
	c.SetAvailable(true)
	closed := p.Size()

	assert.NotPanics(t, func() {
		p.Drawer().SetOpen(true)
		p.Drawer().YieldFocus()
	})
	assert.Greater(t, p.Size().Width, closed.Width)
	assert.Equal(t, closed.Height, p.Size().Height)
}
