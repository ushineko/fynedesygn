package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// ordered builds a panel of three drawn cards, attached to a window.
func ordered(t *testing.T) (*glance.Panel, fyne.Window, []*glance.Card) {
	t.Helper()
	test.NewTempApp(t)
	p := glance.NewPanel(0)
	var cards []*glance.Card
	for _, name := range []string{"Bandwidth", "Cooler", "Usage"} {
		c := glance.NewCard(name)
		c.AddRow(named(name+":row", name))
		c.SetAvailable(true)
		cards = append(cards, c)
	}
	p.Add(cards...)
	w := test.NewTempWindow(t, nil)
	p.Attach(w)
	return p, w, cards
}

// tops is each card's top, in the panel's own order of cards.
func tops(cards ...*glance.Card) []float32 {
	out := make([]float32, len(cards))
	for i, c := range cards {
		out[i] = c.Object().Position().Y
	}
	return out
}

// Spec 060. A new order is drawn, top to bottom, in Stack and in Lines, and
// the panel reports it.
func TestANewOrderIsDrawnInStackAndLines(t *testing.T) {
	for _, a := range []glance.Arrangement{glance.Stack, glance.Lines} {
		p, _, c := ordered(t)
		p.SetArrangement(a)
		require.NoError(t, p.SetOrder(c[2], c[0], c[1]))

		assert.Equal(t, []*glance.Card{c[2], c[0], c[1]}, p.Cards(), "arrangement %d", a)
		y := tops(c[2], c[0], c[1])
		assert.Less(t, y[0], y[1], "arrangement %d: the first card is not on top", a)
		assert.Less(t, y[1], y[2], "arrangement %d: the second card is not above the third", a)
	}
}

// In Grid the cards take their columns in the new order: the layout reads the
// same list the panel reports.
func TestANewOrderIsTheGridsOrder(t *testing.T) {
	p, _, c := ordered(t)
	p.SetArrangement(glance.Grid)
	require.NoError(t, p.SetOrder(c[1], c[2], c[0]))
	assert.Equal(t, []*glance.Card{c[1], c[2], c[0]}, p.Cards())
	stack := p.Content().(*fyne.Container).Objects[1].(*fyne.Container)
	require.Len(t, stack.Objects, 3)
	assert.Same(t, c[1].Object(), stack.Objects[0])
	assert.Same(t, c[2].Object(), stack.Objects[1])
	assert.Same(t, c[0].Object(), stack.Objects[2])
}

// A card keeps everything it had: its rows (the same handles), whether it is
// allowed, its stale mark, and the arrangement.
func TestACardKeepsItsStateAcrossANewOrder(t *testing.T) {
	p, _, c := ordered(t)
	p.SetArrangement(glance.Lines)
	row := c[1].RowByID("Cooler:row")
	require.NotNil(t, row)
	c[0].SetAllowed(false)
	c[2].SetStale(true)

	require.NoError(t, p.SetOrder(c[2], c[1], c[0]))

	assert.Same(t, row, c[1].RowByID("Cooler:row"), "the row was rebuilt")
	assert.False(t, c[0].Drawn(), "a hidden card came back")
	assert.True(t, c[2].Stale(), "a stale card lost its mark")
	assert.Equal(t, glance.Lines, p.Arrangement())
}

// Cards not named keep their order after the ones that are.
func TestCardsNotNamedFollowInTheirOwnOrder(t *testing.T) {
	p, _, c := ordered(t)
	require.NoError(t, p.SetOrder(c[2]))
	assert.Equal(t, []*glance.Card{c[2], c[0], c[1]}, p.Cards())
}

// A card the panel does not hold, or one named twice, is refused, and the
// order stays as it was.
func TestAnOrderNamingAStrangerOrATwiceIsRefused(t *testing.T) {
	p, _, c := ordered(t)
	stranger := glance.NewCard("Elsewhere")

	assert.ErrorIs(t, p.SetOrder(c[1], stranger), glance.ErrCardOrder)
	assert.ErrorIs(t, p.SetOrder(c[1], c[1]), glance.ErrCardOrder)
	assert.Equal(t, c, p.Cards(), "a refused order changed the panel")
}

// A new order is a change of shape and the window is resized for it, once:
// the same cards in another order are the same height. The same order again
// does nothing.
func TestANewOrderKeepsTheWindowsSize(t *testing.T) {
	p, w, c := ordered(t)
	before := w.Canvas().Size()
	require.NoError(t, p.SetOrder(c[2], c[1], c[0]))
	assert.InDelta(t, before.Height, w.Canvas().Size().Height, 1)
	assert.InDelta(t, before.Width, w.Canvas().Size().Width, 1)
	require.NoError(t, p.SetOrder(p.Cards()...), "the order the panel has is no change")
}
