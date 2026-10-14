package glance_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// tallDrawer is an open drawer that scrolls, holding n lines, and the lines.
func tallDrawer(t *testing.T, n int) (*glance.Window, []*glance.Card, []*glance.Row) {
	t.Helper()
	w, cards, _ := drawerWindow(t, glance.Options{Title: "scroll"})
	d := w.Panel().Drawer()
	d.SetScrolling(true)
	rows := make([]*glance.Row, n)
	pieces := make([]glance.Piece, n)
	for i := range rows {
		rows[i] = glance.NewRow(fmt.Sprintf("limit %d", i), glance.NoRate())
		pieces[i] = rows[i]
	}
	d.SetTitle("Usage")
	d.SetContent(pieces...)
	d.SetOpen(true)
	return w, cards, rows
}

// Spec 069 R7, R8. Content taller than the room does not make the window
// taller: the drawer is still the cards' height.
func TestAScrollingDrawerDoesNotSetTheHeight(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "scroll"})
	closed := w.Window().Canvas().Size()
	d := w.Panel().Drawer()
	d.SetScrolling(true)
	pieces := make([]glance.Piece, 80)
	for i := range pieces {
		pieces[i] = glance.NewRow("limit", glance.NoRate())
	}
	d.SetContent(pieces...)
	d.SetOpen(true)
	w.Panel().Resize()
	assert.InDelta(t, closed.Height, w.Window().Canvas().Size().Height, 0.01)
	assert.InDelta(t, closed.Height, d.Frame().Size().Height, 0.01)
	assert.InDelta(t, closed.Height, w.Panel().Content().MinSize().Height, 0.01,
		"the content asks for more height than the cards")
}

/*
R10. The wheel over the drawer scrolls it, and over a card it does not. The
first half is also a canary: Fyne finds the target of a wheel event among the
Scrollable objects under the pointer, and the panel's catcher, which takes
every click (quirk 24), is not one. A Fyne that sent the wheel where it sends
a click would fail here.
*/
func TestTheWheelScrollsTheDrawerAndNotTheCards(t *testing.T) {
	w, cards, rows := tallDrawer(t, 80)
	d, cv := w.Panel().Drawer(), w.Window().Canvas()
	first := middle(rows[0].Object()).Y

	test.Scroll(cv, middle(cards[0].Object()), 0, -60)
	assert.Zero(t, d.Offset(), "the wheel over a card scrolled the drawer")

	test.Scroll(cv, middle(d.Frame()), 0, -60)
	assert.Greater(t, d.Offset(), float32(0), "the wheel over the drawer did nothing")
	assert.Less(t, middle(rows[0].Object()).Y, first, "the content did not move")

	test.Scroll(cv, middle(d.Frame()), 0, 1e6)
	assert.Zero(t, d.Offset(), "the content went past its top")
	test.Scroll(cv, middle(d.Frame()), 0, -1e6)
	end := d.Offset()
	last := rows[len(rows)-1].Object()
	frame := d.Frame()
	assert.LessOrEqual(t, middle(last).Y, fyne.CurrentApp().Driver().AbsolutePositionForObject(frame).Y+frame.Size().Height,
		"the last line cannot be reached")
	test.Scroll(cv, middle(d.Frame()), 0, -60)
	assert.InDelta(t, end, d.Offset(), 0.01, "the content went past its end")
}

// R10. The keys scroll it while no filter is being typed, and a filter being
// typed keeps its keys.
func TestTheKeysScrollTheDrawer(t *testing.T) {
	w, _, _ := tallDrawer(t, 80)
	d := w.Panel().Drawer()
	typed := w.Window().Canvas().OnTypedKey()
	require.NotNil(t, typed, "a drawer that scrolls and has no filter has no keys")

	typed(key(fyne.KeyDown))
	line := d.Offset()
	assert.Greater(t, line, float32(0))
	typed(key(fyne.KeyPageDown))
	assert.Greater(t, d.Offset(), 2*line, "a page is no further than a line")
	typed(key(fyne.KeyUp))
	typed(key(fyne.KeyPageUp))
	assert.Zero(t, d.Offset())
	typed(key(fyne.KeyEnd))
	end := d.Offset()
	assert.Greater(t, end, float32(0))
	typed(key(fyne.KeyDown))
	assert.InDelta(t, end, d.Offset(), 0.01)
	typed(key(fyne.KeyHome))
	assert.Zero(t, d.Offset())

	d.SetOpen(false)
	typed(key(fyne.KeyDown))
	assert.Zero(t, d.Offset(), "a key scrolled a closed drawer")
}

// R11, R12. New content and closing go back to the top; a line set in place
// leaves the content where it is.
func TestTheOffsetIsKeptThroughAReadingAndNotThroughNewContent(t *testing.T) {
	w, _, rows := tallDrawer(t, 80)
	d, cv := w.Panel().Drawer(), w.Window().Canvas()

	test.Scroll(cv, middle(d.Frame()), 0, -90)
	at := d.Offset()
	require.Greater(t, at, float32(0))
	rows[3].SetLabel("another reading")
	w.Panel().Resize()
	assert.InDelta(t, at, d.Offset(), 0.01, "a reading moved what was being read")

	d.SetOpen(false)
	d.SetOpen(true)
	assert.Zero(t, d.Offset(), "the drawer opened where it was left")

	test.Scroll(cv, middle(d.Frame()), 0, -90)
	d.SetContent(rows[0], rows[1])
	assert.Zero(t, d.Offset(), "new content began part of the way down")
}

// R7. Content that fits is drawn the same whether the drawer scrolls or not,
// and has nowhere to scroll to.
func TestContentThatFitsIsDrawnAsWithoutScrolling(t *testing.T) {
	picture := func(scrolling bool) (fyne.Position, fyne.Size, float32) {
		w, _, _ := drawerWindow(t, glance.Options{Title: "scroll"})
		d := w.Panel().Drawer()
		d.SetScrolling(scrolling)
		row := glance.NewRow("limit", glance.NoRate())
		d.SetTitle("Usage")
		d.SetContent(row)
		d.SetOpen(true)
		test.Scroll(w.Window().Canvas(), middle(d.Frame()), 0, -60)
		drv := fyne.CurrentApp().Driver()
		return drv.AbsolutePositionForObject(row.Object()).Subtract(drv.AbsolutePositionForObject(d.Frame())),
			row.Object().Size(), d.Offset()
	}
	at, size, _ := picture(false)
	scrolledAt, scrolledSize, offset := picture(true)
	assert.Equal(t, at, scrolledAt)
	assert.Equal(t, size, scrolledSize)
	assert.Zero(t, offset)
}

// Content given to a drawer that is already open scrolls like content it
// was opened with. The scroller measures what it holds when it is laid out,
// and new content arrives after that.
func TestContentSetWhileOpenScrolls(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "scroll"})
	d := w.Panel().Drawer()
	d.SetContent(glance.NewRow("a list", glance.NoRate()))
	d.SetOpen(true)

	d.SetScrolling(true)
	pieces := make([]glance.Piece, 80)
	for i := range pieces {
		pieces[i] = glance.NewRow("limit", glance.NoRate())
	}
	d.SetContent(pieces...)
	w.Window().Canvas().OnTypedKey()(key(fyne.KeyPageDown))
	assert.Greater(t, d.Offset(), float32(0))
}
