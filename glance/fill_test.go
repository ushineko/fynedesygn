package glance_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// Spec 080: a full row of drawers fills the room left beside it.

// width is an object's width across the window.
func width(d *glance.Drawer) float32 { return d.Frame().Size().Width }

// twoOut is a drawer window with two drawers open on a side.
func twoOut(t *testing.T, side glance.Side) (*glance.Window, *glance.Drawer, *glance.Drawer) {
	t.Helper()
	w, _, _ := drawerWindow(t, glance.Options{Title: "fill"})
	p := w.Panel()
	p.SetSelfMoving(true)
	first, second := p.Drawer(), p.AddDrawer()
	first.SetSide(side)
	first.SetOpen(true)
	second.SetOpen(true)
	return w, first, second
}

// R1, R4. Two drawers and 101 of room on their side at scale 1: the window is
// 101 wider, both drawers are the same width, and the pixel the division
// leaves over stands beyond the outermost.
func TestFillSharesTheRoomEqually(t *testing.T) {
	w, first, second := twoOut(t, glance.Right)
	p := w.Panel()
	before := w.Window().Canvas().Size().Width
	own := width(first)

	require.True(t, p.Fill(0, 101))
	assert.True(t, p.Filled())
	after := w.Window().Canvas().Size().Width
	assert.InDelta(t, math.Ceil(float64(before))+101, math.Ceil(float64(after)), 0.01, "the window did not reach the edge")
	assert.Equal(t, width(first), width(second), "the drawers are not the same width")
	assert.Greater(t, width(first), own)

	end := x(second.Frame()) + width(second)
	pad := math.Ceil(float64(after)) - float64(end)
	assert.GreaterOrEqual(t, pad, 0.0)
	assert.Less(t, pad, 2.0, "the padding is a drawer's share or more")
}

// R4. At a scale of 1.5 the drawers are each the same number of pixels, and
// what the division leaves over is fewer pixels than there are drawers.
func TestFillAtAFractionalScale(t *testing.T) {
	const scale, cards, gap = 1.5, 218.67, 12.6
	reach := float32(32*14) + gap
	for room := float32(1); room < 300; room += 0.37 {
		each, pad, window, ok := glance.PlanFill(scale, cards, 3*reach, gap, room, 3)
		require.True(t, ok, "room %v", room)
		gapPx := int(math.Round(gap * scale))
		cardsPx := int(math.Ceil(cards * scale))
		assert.Equal(t, window, cardsPx+3*gapPx+3*each+pad, "the pixels do not add up at room %v", room)
		assert.GreaterOrEqual(t, pad, 0)
		assert.Less(t, pad, 3, "room %v", room)
		// The window is the one it was and the whole pixels of the room.
		was := int(math.Ceil(float64((cards + 3*reach) * scale)))
		assert.Equal(t, was+int(math.Floor(float64(room*scale)+1e-3)), window)
	}
}

// R1. Nothing to fill: no drawer open, or no whole pixel of room.
func TestFillWithNothingToFill(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "fill"})
	p := w.Panel()
	before := w.Window().Canvas().Size()
	assert.False(t, p.Fill(500, 500), "filled with no drawer open")
	assert.False(t, p.Filled())

	p.Drawer().SetSide(glance.Right)
	p.Drawer().SetOpen(true)
	open := w.Window().Canvas().Size()
	assert.False(t, p.Fill(500, 0.5), "filled less than a pixel, or the other side's room")
	assert.Equal(t, open, w.Window().Canvas().Size())
	assert.NotEqual(t, before, open)
}

// R5, R6. The cards keep their width, place and height through Fill and
// Unfill, on either side.
func TestFillLeavesTheCards(t *testing.T) {
	for _, side := range []glance.Side{glance.Right, glance.Left} {
		w, first, _ := twoOut(t, side)
		p := w.Panel()
		cards, height := p.CardsWidth(), w.Window().Canvas().Size().Height
		// On the left the cards stand off the window's left edge by the
		// drawers, so their place on the screen is the window's right edge
		// less their width: the distance from the right edge is what holds.
		fromRight := w.Window().Canvas().Size().Width - p.CardsOrigin().X

		require.True(t, p.Fill(80, 80))
		assert.InDelta(t, cards, p.CardsWidth(), 0.02, "side %v: the cards changed width", side)
		assert.Equal(t, height, w.Window().Canvas().Size().Height)
		if side == glance.Right {
			assert.Zero(t, p.CardsOrigin().X)
		} else {
			assert.InDelta(t, fromRight, w.Window().Canvas().Size().Width-p.CardsOrigin().X, 0.02)
			// The first drawer opened stands a gap from the cards, the
			// second a gap beyond it, and the padding is at the window's
			// left edge.
			second := p.Outermost()
			assert.InDelta(t, p.CardsOrigin().X-p.Gap(), x(first.Frame())+width(first), 1.01, "the inner drawer is not beside the cards")
			assert.InDelta(t, x(first.Frame())-p.Gap(), x(second.Frame())+width(second), 1.01)
			assert.GreaterOrEqual(t, x(second.Frame()), float32(0))
			assert.Less(t, x(second.Frame()), float32(2), "the padding is not at the far end")
		}

		p.Unfill()
		assert.False(t, p.Filled())
		assert.InDelta(t, cards, p.CardsWidth(), 0.02)
		assert.InDelta(t, first.Reach()-p.Gap(), width(first), 0.01, "side %v: the drawer is not its own width", side)
	}
}

// R7. Closing a drawer while filled ends filling in one resize, and so does
// CloseDrawers.
func TestClosingADrawerEndsFilling(t *testing.T) {
	w, first, second := twoOut(t, glance.Right)
	p := w.Panel()
	third := p.AddDrawer()
	third.SetOpen(true)
	require.True(t, p.Fill(0, 120))

	resizes := p.CountResizes()
	second.SetOpen(false)
	assert.False(t, p.Filled())
	assert.Equal(t, 1, *resizes, "closing a drawer of a filled row resized more than once")
	assert.InDelta(t, first.Reach()-p.Gap(), width(first), 0.01)
	assert.InDelta(t, p.CardsWidth()+2*first.Reach(), w.Window().Canvas().Size().Width, 0.02)

	require.True(t, p.Fill(0, 120))
	p.CloseDrawers()
	assert.False(t, p.Filled())
	assert.InDelta(t, p.CardsWidth(), w.Window().Canvas().Size().Width, 0.02)
}

// R8. A filled drawer given new content stays filled.
func TestNewContentKeepsTheRowFilled(t *testing.T) {
	w, _, second := twoOut(t, glance.Right)
	p := w.Panel()
	require.True(t, p.Fill(0, 60))
	size := w.Window().Canvas().Size()
	second.SetTitle("Memory")
	second.SetContent(glance.NewMeter("used", 40))
	assert.True(t, p.Filled())
	assert.Equal(t, size, w.Window().Canvas().Size())
}

// R11. The Fill button is on the outermost open drawer only, follows it, is
// drawn engaged while filled, and a press calls the toggle.
func TestTheFillButtonIsOnTheOutermost(t *testing.T) {
	w, first, second := twoOut(t, glance.Right)
	p := w.Panel()
	pressed := 0
	p.SetFillable(true, func() { pressed++ })

	_, shown, _ := first.FillButton()
	assert.False(t, shown, "the inner drawer has the button")
	label, shown, active := second.FillButton()
	assert.True(t, shown)
	assert.Equal(t, glance.FillWord, label)
	assert.False(t, active)
	second.PressFill()
	assert.Equal(t, 1, pressed)

	require.True(t, p.Fill(0, 60))
	_, _, active = second.FillButton()
	assert.True(t, active, "the button is not engaged while filled")

	second.SetOpen(false)
	_, shown, _ = first.FillButton()
	assert.True(t, shown, "the button did not follow the outermost")
	_, shown, _ = second.FillButton()
	assert.False(t, shown)

	p.SetFillable(false, nil)
	_, shown, _ = first.FillButton()
	assert.False(t, shown)
}

// R12. DrawersWidth is the window's width less the cards', filled and not.
func TestDrawersWidth(t *testing.T) {
	w, _, _ := twoOut(t, glance.Right)
	p := w.Panel()
	assert.InDelta(t, w.Window().Canvas().Size().Width-p.CardsWidth(), p.DrawersWidth(), 0.02)
	require.True(t, p.Fill(0, 77))
	assert.InDelta(t, w.Window().Canvas().Size().Width-p.CardsWidth(), p.DrawersWidth(), 0.02)
}

// R10. A new face ends filling.
func TestRestyleEndsFilling(t *testing.T) {
	w, _, _ := twoOut(t, glance.Right)
	p := w.Panel()
	require.True(t, p.Fill(0, 50))
	p.Restyle()
	assert.False(t, p.Filled())
}
