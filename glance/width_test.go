package glance_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// Spec 082: a drawer as wide as what it shows.

// em is the panel's text size, read from its first drawer.
func em(p *glance.Panel) float32 {
	return (p.Drawer().Reach() - p.Gap()) / p.Drawer().WidthInEms()
}

// threeWide is a drawer window with drawers of 32, 44 and 38 ems open on a
// side, in that order outwards.
func threeWide(t *testing.T, side glance.Side) (*glance.Window, []*glance.Drawer) {
	t.Helper()
	w, _, _ := drawerWindow(t, glance.Options{Title: "widths"})
	p := w.Panel()
	p.SetSelfMoving(true)
	ds := []*glance.Drawer{p.Drawer(), p.AddDrawer(), p.AddDrawer()}
	ds[0].SetSide(side)
	for i, ems := range []float32{32, 44, 38} {
		ds[i].SetWidthInEms(ems)
		ds[i].SetOpen(true)
	}
	return w, ds
}

// R1, R5. Drawers of different widths stand side by side, each its own
// width and one gap from the next, beyond the cards on either side; the
// window is the cards and the reaches wide, and the cards did not move.
func TestDrawersOfDifferentWidths(t *testing.T) {
	for _, side := range []glance.Side{glance.Right, glance.Left} {
		w, _, _ := drawerWindow(t, glance.Options{Title: "widths"})
		p := w.Panel()
		p.SetSelfMoving(true)
		ds := []*glance.Drawer{p.Drawer(), p.AddDrawer(), p.AddDrawer()}
		ds[0].SetSide(side)
		cardsAt := p.CardsOrigin().X
		fromRight := w.Window().Canvas().Size().Width - cardsAt
		e, gap := em(p), p.Gap()

		var reaches float32
		for i, ems := range []float32{32, 44, 38} {
			ds[i].SetWidthInEms(ems)
			ds[i].SetOpen(true)
			assert.InDelta(t, ems*e, width(ds[i]), 0.01, "%v: drawer %d is not its own width", side, i)
			assert.InDelta(t, ems*e+gap, ds[i].Reach(), 0.01)
			reaches += ds[i].Reach()
		}
		size := w.Window().Canvas().Size().Width
		assert.InDelta(t, p.CardsWidth()+reaches, size, 0.02, "%v: the window is not the cards and the reaches", side)

		cards := p.CardsOrigin().X
		edge := cards + p.CardsWidth()
		if side == glance.Left {
			assert.InDelta(t, fromRight, size-cards, 0.02, "the cards moved")
		} else {
			assert.InDelta(t, cardsAt, cards, 0.02, "the cards moved")
		}
		// Each drawer one gap beyond the last, outwards from the cards.
		for i, d := range ds {
			if side == glance.Right {
				assert.InDelta(t, edge+gap, x(d.Frame()), 0.02, "drawer %d is not a gap beyond the last", i)
				edge = x(d.Frame()) + width(d)
			} else {
				assert.InDelta(t, cards-gap, x(d.Frame())+width(d), 0.02, "drawer %d is not a gap beyond the last", i)
				cards = x(d.Frame())
			}
		}
	}
}

// R3. Set on a closed drawer the width resizes nothing; on an open one to a
// new width it resizes once; to the width it has, not at all.
func TestSettingAWidthIsOneChangeOfShape(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "widths"})
	p := w.Panel()
	d := p.Drawer()
	d.SetSide(glance.Right)

	resizes := p.CountResizes()
	d.SetWidthInEms(40)
	assert.Equal(t, 0, *resizes, "a closed drawer's width resized the window")
	assert.InDelta(t, float32(40), d.WidthInEms(), 0)

	d.SetOpen(true)
	*resizes = 0
	before := w.Window().Canvas().Size().Width
	d.SetWidthInEms(48)
	assert.Equal(t, 1, *resizes, "an open drawer's new width is not one change of shape")
	assert.InDelta(t, before+8*em(p), w.Window().Canvas().Size().Width, 0.02)

	*resizes = 0
	d.SetWidthInEms(48)
	assert.Equal(t, 0, *resizes, "the width it had resized the window")
}

// R2. Nothing the drawer is given changes its width.
func TestContentDoesNotSetTheWidth(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "widths"})
	p := w.Panel()
	d := p.Drawer()
	d.SetWidthInEms(44)
	d.SetOpen(true)
	size := w.Window().Canvas().Size()

	r := glance.NewRow("a name that is a great deal longer than any drawer is wide", "--")
	d.SetTitle("A heading that is a great deal longer than the drawer")
	d.SetContent(r)
	r.Set(glance.Reading{Text: "1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26"})
	assert.InDelta(t, float32(44), d.WidthInEms(), 0)
	assert.Equal(t, size, w.Window().Canvas().Size(), "content resized the window")
}

// R1. A new drawer is the usual width; less than a card's is a card's.
func TestTheWidthsBounds(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "widths"})
	d := w.Panel().AddDrawer()
	assert.InDelta(t, float32(glance.DrawerWidthInEms), d.WidthInEms(), 0)
	d.SetWidthInEms(3)
	assert.InDelta(t, float32(glance.CardWidthInEms), d.WidthInEms(), 0)
}

// R6, R7. Room is the widest drawer that fits, in ems, on the drawers' side
// less a gap, or the larger side's with none open; Fits agrees with it at the
// usual width across a sweep of rooms.
func TestRoomIsTheWidestThatFits(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "widths"})
	p := w.Panel()
	e, gap := em(p), p.Gap()

	assert.InDelta(t, float32(50), p.Room(20, 50*e+gap), 0.001, "none open: not the larger side")
	assert.InDelta(t, float32(50), p.Room(50*e+gap, 20), 0.001, "none open: not the larger side")

	p.Drawer().SetSide(glance.Right)
	p.Drawer().SetOpen(true)
	assert.InDelta(t, float32(40), p.Room(900, 40*e+gap), 0.001, "not the drawers' side")
	assert.LessOrEqual(t, p.Room(900, 0), float32(0), "no room is not zero or less")

	for room := float32(0); room < 1000; room += 3.7 {
		assert.Equal(t, p.Room(0, room) >= glance.DrawerWidthInEms, p.Fits(0, room), "room %v", room)
	}
}

// R8. Filled, drawers of different widths each grow by the same whole number
// of pixels, from their own width; the padding is fewer pixels than drawers;
// unfilled, each is its own width again.
func TestFillWidensEachByTheSame(t *testing.T) {
	const scale, cards, gap = 1.5, 289.33, 8
	own := []float32{32 * 13, 44 * 13, 38 * 13}
	for room := float32(1); room < 300; room += 0.37 {
		widths, pad, window, ok := glance.PlanFillOwn(scale, cards, gap, room, own)
		require.True(t, ok, "room %v", room)
		grown := widths[0] - int(math.Floor(float64(own[0]*scale)+1e-3))
		for i, wpx := range widths {
			assert.Equal(t, grown, wpx-int(math.Floor(float64(own[i]*scale)+1e-3)), "drawer %d grew by another amount at room %v", i, room)
		}
		assert.GreaterOrEqual(t, grown, 0)
		assert.GreaterOrEqual(t, pad, 0)
		assert.Less(t, pad, len(own), "room %v", room)
		gapPx := int(math.Round(gap * scale))
		sum := 0
		for _, wpx := range widths {
			sum += gapPx + wpx
		}
		assert.Equal(t, window, int(math.Ceil(cards*scale))+sum+pad, "the pixels do not add up at room %v", room)
	}

	for _, side := range []glance.Side{glance.Right, glance.Left} {
		w, ds := threeWide(t, side)
		p := w.Panel()
		e := em(p)
		require.True(t, p.Fill(80, 80))
		for i, d := range ds {
			assert.Greater(t, width(d), d.WidthInEms()*e, "%v: drawer %d did not grow", side, i)
			assert.InDelta(t, width(ds[0])-ds[0].WidthInEms()*e, width(d)-d.WidthInEms()*e, 0.02, "%v: drawer %d grew by another amount", side, i)
		}
		p.Unfill()
		for i, d := range ds {
			assert.InDelta(t, d.WidthInEms()*e, width(d), 0.01, "%v: drawer %d is not its own width again", side, i)
		}
	}
}

// R9. A new width on a filled row's drawer ends filling in one resize.
func TestANewWidthEndsFilling(t *testing.T) {
	w, ds := threeWide(t, glance.Right)
	p := w.Panel()
	require.True(t, p.Fill(0, 120))
	resizes := p.CountResizes()
	ds[1].SetWidthInEms(36)
	assert.False(t, p.Filled())
	assert.Equal(t, 1, *resizes, "a new width on a filled row resized more than once")
	assert.InDelta(t, 36*em(p), width(ds[1]), 0.01)
}

// R4. A larger text size keeps each drawer's ems.
func TestRestyleKeepsTheEms(t *testing.T) {
	w, ds := threeWide(t, glance.Right)
	p := w.Panel()
	p.Restyle()
	for i, ems := range []float32{32, 44, 38} {
		assert.InDelta(t, ems, ds[i].WidthInEms(), 0, "drawer %d", i)
		assert.InDelta(t, ems*em(p), width(ds[i]), 0.01, "drawer %d", i)
	}
}

// R10. Available is the drawer's own width less its margins, and its filled
// width less the same while filled.
func TestAvailableIsTheShownWidth(t *testing.T) {
	w, ds := threeWide(t, glance.Right)
	p := w.Panel()
	margins := width(ds[0]) - ds[0].Available().Width
	require.Positive(t, margins)
	for i, d := range ds {
		assert.InDelta(t, width(d)-margins, d.Available().Width, 0.01, "drawer %d", i)
	}
	require.True(t, p.Fill(0, 120))
	for i, d := range ds {
		assert.InDelta(t, width(d)-margins, d.Available().Width, 0.02, "drawer %d, filled", i)
	}
}
