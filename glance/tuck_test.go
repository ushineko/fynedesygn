package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// Spec 081: a panel tucks into a square at its corner.

// square is the tucked window's side at the panel's text size.
func square(w *glance.Window) float32 {
	d := w.Panel().Drawer()
	text := (d.Reach() - w.Panel().Gap()) / glance.DrawerWidthInEms
	return glance.TuckSizeInEms * text
}

// R1, R5. Tuck makes the window the square and shows it; Untuck gives back
// the size it had, drawers open.
func TestTuckAndBack(t *testing.T) {
	w, first, second := twoOut(t, glance.Right)
	p := w.Panel()
	p.SetCornerHeld(true)
	open := w.Window().Canvas().Size()
	cards := p.CardsWidth()

	require.True(t, p.Tuck(glance.Left))
	assert.True(t, p.Tucked())
	s := square(w)
	assert.Equal(t, fyne.NewSize(s, s), w.Window().Canvas().Size(), "the window is not the square")
	assert.True(t, p.TuckSquare().Visible())
	assert.False(t, p.Body().Visible(), "the cards are drawn under the square")

	p.Untuck()
	assert.False(t, p.Tucked())
	assert.Equal(t, open, w.Window().Canvas().Size(), "untucking did not give the size back")
	assert.False(t, p.TuckSquare().Visible())
	assert.InDelta(t, cards, p.CardsWidth(), 0.01)
	assert.True(t, first.Open())
	assert.True(t, second.Open())
}

// R3. The corner: away from open drawers, and with none on the side with
// less room.
func TestTheCornerIsTheScreenEdges(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "tuck"})
	p := w.Panel()
	assert.Equal(t, glance.Right, p.TuckCorner(500, 20), "the panel against the right edge tucks left")
	assert.Equal(t, glance.Left, p.TuckCorner(20, 500))

	p.SetSelfMoving(true)
	p.Drawer().SetSide(glance.Left)
	p.Drawer().SetOpen(true)
	assert.Equal(t, glance.Right, p.TuckCorner(20, 500), "the corner is on the drawers' side")
	p.Drawer().SetOpen(false)
	p.Drawer().SetSide(glance.Right)
	p.Drawer().SetOpen(true)
	assert.Equal(t, glance.Left, p.TuckCorner(500, 20))
}

// R4. Drawers open before tucking are open after, in the same order, with
// the same content.
func TestTuckKeepsTheDrawers(t *testing.T) {
	w, first, second := twoOut(t, glance.Right)
	p := w.Panel()
	p.SetCornerHeld(true)
	first.SetTitle("CPU")
	second.SetTitle("Memory")
	require.True(t, p.Tuck(glance.Left))
	p.Untuck()
	assert.Equal(t, []*glance.Drawer{first, second}, []*glance.Drawer{p.Drawers()[0], p.Outermost()})
	assert.Equal(t, "CPU", first.Title())
	assert.Equal(t, "Memory", second.Title())
	assert.Equal(t, 2, p.Opened())
}

// R7. A reading's Resize while tucked keeps the square; so does a drawer
// opening out of sight.
func TestAReadingDoesNotUntuck(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "tuck"})
	p := w.Panel()
	p.SetCornerHeld(true)
	require.True(t, p.Tuck(glance.Right))
	s := w.Window().Canvas().Size()
	cards[0].AddRow(named("more", "more"))
	p.Resize()
	assert.Equal(t, s, w.Window().Canvas().Size())
	p.Drawer().SetOpen(true)
	assert.Equal(t, s, w.Window().Canvas().Size(), "a drawer opened while tucked grew the window")
}

// R6. OnTuck is told both ways.
func TestOnTuck(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "tuck"})
	p := w.Panel()
	p.SetCornerHeld(true)
	var got []bool
	p.SetOnTuck(func(on bool) { got = append(got, on) })
	p.Tuck(glance.Right)
	p.Untuck()
	assert.Equal(t, []bool{true, false}, got)
}

// R9. A window that cannot move itself and whose corner nobody holds does
// not tuck.
func TestNoTuckWithoutAHeldCorner(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "tuck"})
	p := w.Panel()
	p.SetSelfMoving(false)
	before := w.Window().Canvas().Size()
	assert.False(t, p.CanTuck())
	assert.False(t, p.Tuck(glance.Right))
	assert.False(t, p.Tucked())
	assert.Equal(t, before, w.Window().Canvas().Size())
	p.SetCornerHeld(true)
	assert.True(t, p.CanTuck())
}

// R10, R5. The glyph shows only while the pointer is over the cards, a tap
// on it asks to tuck, and a tap on the square untucks.
func TestTheGlyphAndTheSquare(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "tuck"})
	p := w.Panel()
	p.SetCornerHeld(true)
	cv := w.Window().Canvas()
	asked := 0
	tapped := 0
	p.OnTapped = func(glance.Tap) { tapped++ }
	p.SetTuckable(true, func() { asked++; p.Tuck(glance.Right) })

	assert.False(t, p.TuckGlyph().Visible())
	test.MoveMouse(cv, middle(cards[0].Object()))
	require.True(t, p.TuckGlyph().Visible(), "the glyph is not shown over the cards")
	// It is in the cards' top-right corner, inside the window.
	g := fyne.CurrentApp().Driver().AbsolutePositionForObject(p.TuckGlyph())
	assert.Greater(t, g.X, p.CardsWidth()/2)
	assert.LessOrEqual(t, g.X+p.TuckGlyph().Size().Width, p.CardsWidth())
	assert.Less(t, g.Y, cards[0].Object().Size().Height/2)

	test.TapCanvas(cv, middle(p.TuckGlyph()))
	assert.Equal(t, 1, asked)
	assert.Zero(t, tapped, "the tap on the glyph was a tap on the card too")
	require.True(t, p.Tucked())
	assert.False(t, p.TuckGlyph().Visible())

	test.TapCanvas(cv, fyne.NewPos(5, 5))
	assert.False(t, p.Tucked(), "a tap on the square did not untuck")
	assert.Zero(t, tapped)

	p.SetTuckable(false, nil)
	test.MoveMouse(cv, middle(cards[1].Object()))
	assert.False(t, p.TuckGlyph().Visible())
}

// R10. The glyph is the cards': off them, over a drawer, it goes; a tap on it
// is not a tap on the card under it.
func TestTheGlyphIsOnlyOverTheCards(t *testing.T) {
	w, cards, _ := drawerWindow(t, glance.Options{Title: "tuck"})
	p := w.Panel()
	cv := w.Window().Canvas()
	asked, tapped := 0, 0
	p.OnTapped = func(glance.Tap) { tapped++ }
	p.SetTuckable(true, func() { asked++ })
	p.Drawer().SetSide(glance.Right)
	p.Drawer().SetOpen(true)

	test.MoveMouse(cv, middle(cards[0].Object()))
	require.True(t, p.TuckGlyph().Visible())
	test.MoveMouse(cv, middle(p.Drawer().Frame()))
	assert.False(t, p.TuckGlyph().Visible(), "the glyph stayed with the pointer over the drawer")

	test.MoveMouse(cv, middle(cards[0].Object()))
	test.TapCanvas(cv, middle(p.TuckGlyph()))
	assert.Equal(t, 1, asked)
	assert.Zero(t, tapped, "the tap on the glyph was a tap on the card under it")
}

// R5. A width the person dragged survives tucking: the square's width is not
// read as theirs.
func TestADraggedWidthSurvivesTucking(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "tuck", Resizable: true})
	p := w.Panel()
	p.SetCornerHeld(true)
	wide := p.Size().Width + 60
	w.Window().Resize(fyne.NewSize(wide, p.Size().Height))
	p.Resize()
	require.InDelta(t, wide, w.Window().Canvas().Size().Width, 0.01)

	p.Tuck(glance.Right)
	p.Untuck()
	assert.InDelta(t, wide, w.Window().Canvas().Size().Width, 0.01, "the dragged width was lost")
}
