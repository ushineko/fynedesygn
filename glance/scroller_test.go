package glance_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// scrollerWindow is a panel with one card: a row, a scroller of n rows that
// shows three, and a sparkline under it.
func scrollerWindow(t *testing.T, n int) (*glance.Window, *glance.Card, *glance.Scroller, *glance.Sparkline) {
	t.Helper()
	w := newTestWindow(t, glance.Options{Title: "scroller"})
	w.Window().SetPadded(false)
	c := glance.NewCard("Network")
	c.AddRow(named("eno2", "eno2"))
	s := glance.NewScroller(3)
	for i := range n {
		s.AddRow(named(fmt.Sprintf("vpn%d", i), fmt.Sprintf("vpn %d", i)))
	}
	spark := glance.NewSparkline(10)
	c.Add(s)
	c.AddObject(spark)
	w.Panel().Add(c)
	c.SetAvailable(true)
	w.Window().Show()
	w.Panel().Resize()
	return w, c, s, spark
}

// top is an object's top edge on the canvas.
func top(o fyne.CanvasObject) float32 {
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(o).Y
}

// pitch is the distance from one of a scroller's rows to the next.
func pitch(s *glance.Scroller) float32 {
	return s.Rows()[0].Object().MinSize().Height + theme.Padding()
}

// Spec 086 R2. Held to three rows, the run is three rows high however many it
// holds, and the card does not grow when a row comes past the third.
func TestAScrollerIsNeverTallerThanTheRowsItShows(t *testing.T) {
	w, c, s, _ := scrollerWindow(t, 3)
	three := c.Object().MinSize().Height
	window := w.Window().Canvas().Size().Height
	assert.InDelta(t, 3*pitch(s)-theme.Padding(), s.Object().MinSize().Height, 0.01)

	s.AddRow(named("vpn3", "vpn 3"), named("vpn4", "vpn 4"))
	w.Panel().Resize()
	assert.InDelta(t, three, c.Object().MinSize().Height, 0.01, "a fourth and fifth row grew the card")
	assert.InDelta(t, window, w.Window().Canvas().Size().Height, 0.01, "a fourth and fifth row grew the window")
}

// Spec 086 R2. With fewer rows than it may show, the run is as tall as its rows.
func TestAShortScrollerIsTheHeightOfItsRows(t *testing.T) {
	_, _, s, _ := scrollerWindow(t, 2)
	assert.InDelta(t, 2*pitch(s)-theme.Padding(), s.Object().MinSize().Height, 0.01)
}

// Spec 086 R3. A turn of the wheel over the run moves it a row, stops at the
// last row, and comes back to the first. Over the rest of the card it does
// nothing to the run.
func TestTheWheelMovesAScrollerByWholeRows(t *testing.T) {
	w, c, s, _ := scrollerWindow(t, 5)
	cv := w.Window().Canvas()
	over := middle(s.Object())

	wheel(t, cv, over, -10)
	assert.Equal(t, 1, s.First())
	assert.InDelta(t, pitch(s), s.Offset(), 0.01, "the run stopped between rows")
	assert.InDelta(t, top(s.Object()), top(s.Rows()[1].Object()), 0.01,
		"the second row is not at the top of the run")

	for range 5 {
		wheel(t, cv, over, -10)
	}
	assert.Equal(t, 2, s.First(), "the run scrolled past its last row")

	for range 5 {
		wheel(t, cv, over, 10)
	}
	assert.Equal(t, 0, s.First())

	wheel(t, cv, middle(c.Rows()[0].Object()), -10)
	assert.Equal(t, 0, s.First(), "the wheel over a row outside the run scrolled it")
}

// Spec 086 R3. A notch on a scaled desktop carries nearly two rows' travel
// (Fyne's 25 times 1.5, on the desk this was found on) and still moves one
// row; only a spin Fyne accelerated moves more.
func TestANotchIsOneRowWhateverItCarries(t *testing.T) {
	w, _, s, _ := scrollerWindow(t, 5)
	cv := w.Window().Canvas()
	over := middle(s.Object())

	wheel(t, cv, over, -37.5)
	assert.Equal(t, 1, s.First(), "one notch moved more than a row")

	wheel(t, cv, over, -25*125)
	assert.Equal(t, 2, s.First(), "an accelerated spin did not run to the end")
}

// Spec 086 R5. A reading set in place leaves the run where it is; a row taken
// away above the top keeps the top row at the top; the top row taken away
// leaves the run within what there is.
func TestAScrollerKeepsItsPlaceAsItsRowsChange(t *testing.T) {
	w, _, s, _ := scrollerWindow(t, 6)
	s.ScrollTo(2)
	at := s.Rows()[2]

	at.Set(glance.Measured("41.0 °C"))
	assert.Equal(t, 2, s.First(), "a reading moved the run")

	require.True(t, s.RemoveRow("vpn0"))
	w.Panel().Resize()
	assert.Equal(t, at, s.Rows()[s.First()], "a row taken away above moved another to the top")

	s.ScrollTo(99)
	assert.Equal(t, 2, s.First(), "ScrollTo went past the last row")
	last := s.Rows()[len(s.Rows())-1]
	require.True(t, s.RemoveRow(last.ID()))
	w.Panel().Resize()
	assert.Equal(t, 1, s.First(), "the run is past its rows after one was taken away")
}

// Spec 086 R6. A click lands on the row under the pointer after scrolling,
// and a row scrolled out of sight is never the one clicked.
func TestAClickOnAScrolledRunLandsOnTheRowUnderThePointer(t *testing.T) {
	w, c, s, _ := scrollerWindow(t, 5)
	var got []glance.Tap
	w.Panel().OnTapped = func(tap glance.Tap) { got = append(got, tap) }
	cv := w.Window().Canvas()
	s.ScrollTo(2)

	test.TapCanvas(cv, fyne.NewPos(middle(s.Object()).X, top(s.Object())+1))
	require.Len(t, got, 1)
	assert.Equal(t, s.Rows()[2], got[0].Row)

	test.TapCanvas(cv, middle(c.Rows()[0].Object()))
	require.Len(t, got, 2)
	assert.Equal(t, c.Rows()[0], got[1].Row, "a click on the row above the run went to a row scrolled under it")

	// The first row is scrolled up under the heading, out of sight.
	test.TapCanvas(cv, middle(s.Rows()[0].Object()))
	require.Len(t, got, 3)
	assert.Nil(t, got[2].Row, "a click on the heading went to a row scrolled out of sight under it")

	// And the last rows lie under the plot until they are scrolled to.
	s.ScrollTo(0)
	test.TapCanvas(cv, middle(s.Rows()[4].Object()))
	require.Len(t, got, 4)
	assert.Nil(t, got[3].Row, "a click on the plot went to a row not yet scrolled to")
}

// Spec 086 R1. What is added after the run stays under it and does not move
// when it scrolls.
func TestWhatIsUnderAScrollerDoesNotScroll(t *testing.T) {
	w, _, s, spark := scrollerWindow(t, 5)
	before := top(spark)
	wheel(t, w.Window().Canvas(), middle(s.Object()), -10)
	require.Equal(t, 1, s.First())
	assert.InDelta(t, before, top(spark), 0.01)
	assert.Greater(t, top(spark), top(s.Object())+s.Object().Size().Height-0.01)
}

// Spec 086 R7. Drawn as one line per reading, the run is still held to its
// rows and is not tucked away with the plot.
func TestAScrollerStaysInLines(t *testing.T) {
	w, c, s, spark := scrollerWindow(t, 5)
	c.SetArrangement(glance.Lines)
	w.Panel().Resize()
	assert.True(t, s.Object().Visible())
	assert.False(t, spark.Visible())
	assert.InDelta(t, 3*pitch(s)-theme.Padding(), s.Object().MinSize().Height, 0.01)
}

// A stale card dims the rows in its run, and a row added while it is stale.
func TestAStaleCardDimsItsScrolledRows(t *testing.T) {
	_, c, s, _ := scrollerWindow(t, 2)
	c.SetStale(true)
	assert.True(t, s.Rows()[0].Reading().Stale)
	s.AddRow(named("late", "late"))
	assert.True(t, s.RowByID("late").Reading().Stale)
	c.SetStale(false)
	assert.False(t, s.Rows()[0].Reading().Stale)
}
