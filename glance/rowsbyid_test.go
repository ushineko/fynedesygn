package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// named is a row with an ID, for these tests.
func named(id, label string) *glance.Row {
	r := glance.NewRow(label, glance.NoQuantity("°C", 2))
	r.SetID(id)
	return r
}

// ids are the card's rows' IDs, in order.
func ids(c *glance.Card) []string {
	out := make([]string, 0, len(c.Rows()))
	for _, r := range c.Rows() {
		out = append(out, r.ID())
	}
	return out
}

// column is the card's body as it is drawn: each object's row ID, or "plot"
// for the object the test added after the rows. It reads the drawn tree, so it
// checks where InsertRow put the row on screen and not only in Rows().
func column(t *testing.T, c *glance.Card, rows []*glance.Row, plot fyne.CanvasObject) []string {
	t.Helper()
	body := bodyOf(t, c)
	out := make([]string, 0, len(body.Objects))
	for _, o := range body.Objects {
		if o == plot {
			out = append(out, "plot")
			continue
		}
		for _, r := range rows {
			if r.Object() == o {
				out = append(out, r.ID())
			}
		}
	}
	return out
}

// bodyOf finds the container holding the card's rows: the one whose objects
// include the first row's.
func bodyOf(t *testing.T, c *glance.Card) *fyne.Container {
	t.Helper()
	require.NotEmpty(t, c.Rows())
	want := c.Rows()[0].Object()
	var found *fyne.Container
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		ct, ok := o.(*fyne.Container)
		if !ok || found != nil {
			return
		}
		for _, child := range ct.Objects {
			if child == want {
				found = ct
				return
			}
		}
		for _, child := range ct.Objects {
			walk(child)
		}
	}
	walk(c.Object())
	require.NotNil(t, found, "no container in the card holds its rows")
	return found
}

// Spec 056 R2. A row inserted between two takes its place in Rows and on
// screen, and the rows after it keep their names.
func TestARowIsInsertedWhereItIsAsked(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	cpu, coolant := named("cpu", "CPU"), named("coolant", "Coolant")
	c.AddRow(cpu, coolant)

	gpu := named("gpu", "GPU")
	require.NoError(t, c.InsertRow(1, gpu))

	assert.Equal(t, []string{"cpu", "gpu", "coolant"}, ids(c))
	assert.Equal(t, []string{"cpu", "gpu", "coolant"}, column(t, c, []*glance.Row{cpu, gpu, coolant}, nil))

	top := named("top", "Top")
	require.NoError(t, c.InsertRow(0, top))
	assert.Equal(t, []string{"top", "cpu", "gpu", "coolant"}, ids(c))
}

// Spec 056 R3. A row inserted after the plot was added is drawn above the
// plot, not under it: the bug the slack rows in hayami were built to avoid.
func TestARowInsertedAtTheEndIsDrawnAboveThePlot(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	cpu := named("cpu", "CPU")
	c.AddRow(cpu)
	plot := canvas.NewRectangle(nil)
	c.AddObject(plot)

	reason := named("reason:coolant", "Coolant")
	require.NoError(t, c.InsertRow(len(c.Rows()), reason))

	assert.Equal(t, []string{"cpu", "reason:coolant", "plot"},
		column(t, c, []*glance.Row{cpu, reason}, plot))
}

// Spec 056 R3. With no rows yet, an inserted row goes to the top of the card,
// above whatever was added.
func TestTheFirstRowOfACardWithOnlyAPlotGoesAboveIt(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	plot := canvas.NewRectangle(nil)
	c.AddObject(plot)

	first := named("first", "First")
	require.NoError(t, c.InsertRow(0, first))

	assert.Equal(t, []string{"first", "plot"}, column(t, c, []*glance.Row{first}, plot))
}

// Spec 056 R4. A row is taken out by its ID, from Rows and from the screen; a
// name the card does not hold, and the empty name, take out nothing.
func TestARowIsRemovedByItsID(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	cpu, gpu := named("cpu", "CPU"), named("gpu", "GPU")
	c.AddRow(cpu, gpu)
	plot := canvas.NewRectangle(nil)
	c.AddObject(plot)

	assert.True(t, c.RemoveRow("cpu"))
	assert.Equal(t, []string{"gpu"}, ids(c))
	assert.Equal(t, []string{"gpu", "plot"}, column(t, c, []*glance.Row{cpu, gpu}, plot))

	assert.False(t, c.RemoveRow("cpu"), "a row was removed twice")
	assert.False(t, c.RemoveRow(""), "the empty ID found a row")
	assert.Equal(t, []string{"gpu"}, ids(c))
}

// Spec 056 R5. A row is found by its ID; an unknown or empty ID finds nothing.
func TestARowIsFoundByItsID(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	cpu := named("cpu", "CPU")
	anon := glance.NewRow("Fans", glance.NoCount(glance.NumberWidth, "rpm"))
	c.AddRow(cpu, anon)

	assert.Same(t, cpu, c.RowByID("cpu"))
	assert.Nil(t, c.RowByID("gpu"))
	assert.Nil(t, c.RowByID(""), "the empty ID found a row without one")
}

// Spec 056 R1. Two rows cannot answer to one name: InsertRow refuses the
// second and leaves the card as it was, AddRow panics, and rows without an ID
// never collide.
func TestACardRefusesASecondRowWithTheSameID(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	c.AddRow(named("cpu", "CPU"))

	err := c.InsertRow(1, named("cpu", "CPU again"))
	require.ErrorIs(t, err, glance.ErrRowID)
	assert.Equal(t, []string{"cpu"}, ids(c), "a refused row was put in the card")

	assert.Panics(t, func() { c.AddRow(named("cpu", "CPU again")) })

	c.AddRow(glance.NewRow("A", "–"), glance.NewRow("B", "–"))
	assert.Len(t, c.Rows(), 3, "two rows without an ID collided")
}

// Spec 056 R2. A position outside the rows is refused and changes nothing.
func TestAPositionOutsideTheRowsIsRefused(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	c.AddRow(named("cpu", "CPU"))

	require.ErrorIs(t, c.InsertRow(2, named("gpu", "GPU")), glance.ErrRowIndex)
	require.ErrorIs(t, c.InsertRow(-1, named("gpu", "GPU")), glance.ErrRowIndex)
	assert.Equal(t, []string{"cpu"}, ids(c))
}

// Spec 056 R6. A row whose label column is pinned keeps its width wherever it
// ends up: the pin is the row's, so a row inserted above it, or found again by
// its ID, does not move the column a hardware name was pinned to.
func TestAPinnedLabelColumnFollowsItsRow(t *testing.T) {
	test.NewTempApp(t)
	const pin float32 = 80
	c := glance.NewCard("Cooler")
	gpu := glance.NewRowWidth("RTX 3060 Ti", glance.NoQuantity("°C", 2), pin)
	gpu.SetID("gpu:0")
	c.AddRow(gpu)

	require.NoError(t, c.InsertRow(0, named("cpu", "CPU")))

	found := c.RowByID("gpu:0")
	require.NotNil(t, found)
	box, ok := found.Object().(*fyne.Container)
	require.True(t, ok)
	require.NotEmpty(t, box.Objects)
	assert.InDelta(t, pin, box.Objects[0].MinSize().Width, 0.01,
		"the pinned label column moved when a row was inserted above it")
}

// Spec 056 R7. A row inserted into a card showing last-known values is dimmed
// with the rest, so the card does not draw one row as current among stale ones.
func TestARowInsertedIntoAStaleCardIsDimmed(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	c.AddRow(named("cpu", "CPU"))
	c.SetStale(true)

	gpu := named("gpu", "GPU")
	require.NoError(t, c.InsertRow(1, gpu))

	assert.True(t, gpu.Reading().Stale, "the inserted row was drawn as current in a stale card")
}

// Spec 056 R8. Inserting a row grows the card by the row and nothing else,
// and removing it gives the height back: a change of shape the panel's Resize
// follows, never a change caused by a value.
func TestInsertingAndRemovingARowMovesOnlyTheCardsHeight(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	c.AddRow(named("cpu", "CPU"))
	c.AddObject(canvas.NewRectangle(nil))
	c.SetAvailable(true)
	w := test.NewWindow(c.Object())
	t.Cleanup(w.Close)
	before := c.Object().MinSize()

	// The same label as the row above, so the only thing that changes is the
	// number of rows: a wider label would rightly widen the card.
	gpu := named("gpu", "CPU")
	require.NoError(t, c.InsertRow(1, gpu))
	grown := c.Object().MinSize()
	assert.Greater(t, grown.Height, before.Height, "the card did not grow by the row")
	assert.InDelta(t, before.Width, grown.Width, 0.01, "a short row widened the card")

	require.True(t, c.RemoveRow("gpu"))
	assert.InDelta(t, before.Height, c.Object().MinSize().Height, 0.01,
		"removing the row did not give its height back")
}
