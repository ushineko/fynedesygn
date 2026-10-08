package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/glance"
)

// textOf finds the drawn text s under o, or nil.
func textOf(o fyne.CanvasObject, s string) *canvas.Text {
	switch v := o.(type) {
	case *canvas.Text:
		if v.Text == s {
			return v
		}
	case *fyne.Container:
		for _, c := range v.Objects {
			if t := textOf(c, s); t != nil {
				return t
			}
		}
	}
	return nil
}

// shown is whether o and every container above it under root are visible.
func shown(root, o fyne.CanvasObject) bool {
	var walk func(n fyne.CanvasObject) (found, visible bool)
	walk = func(n fyne.CanvasObject) (bool, bool) {
		if n == o {
			return true, n.Visible()
		}
		if c, ok := n.(*fyne.Container); ok {
			for _, child := range c.Objects {
				if f, v := walk(child); f {
					return true, v && n.Visible()
				}
			}
		}
		return false, false
	}
	_, v := walk(root)
	return v
}

// at is where o is drawn, in the window.
func at(o fyne.CanvasObject) fyne.Position {
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(o)
}

// sameLine is whether two objects share a line: their vertical extents
// overlap.
func sameLine(a, b fyne.CanvasObject) bool {
	pa, pb := at(a), at(b)
	return pa.Y < pb.Y+b.Size().Height && pb.Y < pa.Y+a.Size().Height
}

// lined is a card holding a row, a meter, a grid of two cells and a plot, laid
// out in a window 420 wide.
type lined struct {
	card  *glance.Card
	row   *glance.Row
	meter *glance.Meter
	cells []*glance.Cell
	plot  *glance.Sparkline
	win   fyne.Window
}

func newLined(t *testing.T) lined {
	t.Helper()
	test.NewTempApp(t)
	c := glance.NewCard("Usage")
	row := named("cpu", "CPU")
	c.AddRow(row)
	m := glance.NewMeter("5h", 0)
	m.Set(0.42, glance.Percent(42), fd.StatusGood)
	m.SetTrailing("in 2h")
	m.SetStats("used", "left")
	grid := glance.NewCellGrid()
	mouse, keys := glance.NewCell("Basilisk", glance.NoPercent()), glance.NewCell("F75", glance.NoPercent())
	mouse.Set(glance.Known(glance.Percent(77), fd.StatusGood))
	mouse.SetNote("Discharging")
	mouse.SetBar(0.77, fd.StatusGood)
	keys.Set(glance.Known(glance.Percent(100), fd.StatusGood))
	grid.Add(mouse, keys)
	c.Add(m, grid)
	plot := glance.NewSparkline(10)
	c.AddObject(plot)
	c.SetAvailable(true)
	w := test.NewTempWindow(t, c.Object())
	w.Resize(fyne.NewSize(420, 600))
	return lined{card: c, row: row, meter: m, cells: []*glance.Cell{mouse, keys}, plot: plot, win: w}
}

// relayout lays the card out again at the window's width, as a panel's Resize
// does after a change of shape.
func (l lined) relayout() {
	l.win.Resize(fyne.NewSize(420, 600))
	l.win.Content().Refresh()
}

// Spec 057 R2. Lines leaves the heading and the plot out, and the row stays.
func TestLinesLeavesOutTheHeadingAndThePlot(t *testing.T) {
	l := newLined(t)
	root := l.card.Object()
	title := textOf(root, "Usage")
	require.NotNil(t, title)
	require.True(t, shown(root, title))
	require.True(t, l.plot.Visible())

	l.card.SetArrangement(glance.Lines)
	l.relayout()

	assert.False(t, shown(root, title), "the heading is drawn in Lines")
	assert.False(t, l.plot.Visible(), "the plot is drawn in Lines")
	assert.True(t, shown(root, l.row.Object()), "the row went with the heading")
}

// Spec 057 R3. A meter in Lines is one line: label at the left, figures at the
// right, the bar between them on the same line; the stats row is left out.
func TestAMeterInLinesIsOneLine(t *testing.T) {
	l := newLined(t)
	l.card.SetArrangement(glance.Lines)
	l.relayout()

	root := l.meter.Object()
	label, caption, trailing := textOf(root, "5h"), textOf(root, glance.Percent(42)), textOf(root, "in 2h")
	require.NotNil(t, label)
	require.NotNil(t, caption)
	require.NotNil(t, trailing)
	assert.Nil(t, textOf(root, "used"), "the stats row is drawn in Lines")

	assert.True(t, sameLine(label, caption), "the caption is not on the label's line")
	assert.True(t, sameLine(label, trailing), "the trailing value is not on the label's line")
	assert.Less(t, at(label).X, at(caption).X)
	assert.Less(t, at(caption).X, at(trailing).X, "the trailing value is not right of the caption")
	right := at(root).X + root.Size().Width
	assert.InDelta(t, right, at(trailing).X+trailing.Size().Width, 1, "the figures are not against the right edge")
}

// Spec 057 R4. Cells in Lines are one per line: the name at the left, the
// reading at the right on the same line, and the next cell under it.
func TestCellsInLinesAreOnePerLine(t *testing.T) {
	l := newLined(t)
	// Wide enough for two cells drawn as lines side by side, so one per line
	// is the arrangement's doing and not the width's.
	l.win.Resize(fyne.NewSize(1200, 600))
	mouse, keys := l.cells[0].Object(), l.cells[1].Object()
	require.True(t, sameLine(mouse, keys), "two short cells should start side by side in a 420 card")

	l.card.SetArrangement(glance.Lines)
	l.win.Resize(fyne.NewSize(1200, 600))
	l.win.Content().Refresh()

	assert.False(t, sameLine(mouse, keys), "two cells share a line in Lines")
	assert.InDelta(t, at(mouse).X, at(keys).X, 1, "the cells do not start at the same x")
	name, value := textOf(mouse, "Basilisk"), textOf(mouse, glance.Percent(77))
	require.NotNil(t, name)
	require.NotNil(t, value)
	assert.True(t, sameLine(name, value), "a cell's name and reading are not on one line")
	assert.Less(t, at(name).X, at(value).X)
}

// Spec 057 R5. Leaving Lines puts back what it hid, and only that: an object
// the consumer had hidden itself stays hidden.
func TestLeavingLinesPutsBackWhatItHid(t *testing.T) {
	l := newLined(t)
	root := l.card.Object()
	mine := glance.NewSparkline(5)
	l.card.AddObject(mine)
	mine.Hide()

	l.card.SetArrangement(glance.Lines)
	l.card.SetArrangement(glance.Stack)
	l.relayout()

	assert.True(t, shown(root, textOf(root, "Usage")), "the heading did not come back")
	assert.True(t, l.plot.Visible(), "the plot did not come back")
	assert.False(t, mine.Visible(), "an object its consumer hid was shown")
	assert.NotNil(t, textOf(l.meter.Object(), "used"), "the meter's stats row did not come back")
	assert.True(t, sameLine(l.cells[0].Object(), l.cells[1].Object()), "the cells did not go back side by side")
}

// Spec 057 R6. Rows keep their IDs and handles across a switch (spec 056),
// and a plot added while in Lines stays out until Lines is left.
func TestRowsAndLateObjectsSurviveASwitch(t *testing.T) {
	l := newLined(t)
	l.card.SetArrangement(glance.Lines)
	assert.Same(t, l.row, l.card.RowByID("cpu"))

	late := glance.NewSparkline(5)
	l.card.AddObject(late)
	assert.False(t, late.Visible(), "a plot added in Lines is drawn")

	l.card.SetArrangement(glance.Grid)
	assert.True(t, late.Visible(), "a plot added in Lines did not come back")
	assert.Same(t, l.row, l.card.RowByID("cpu"))
}

// Spec 057 R1. A panel hands its arrangement to every card, including one
// added after it was set.
func TestAPanelHandsLinesToItsCards(t *testing.T) {
	test.NewTempApp(t)
	p := glance.NewPanel(0)
	first := glance.NewCard("First")
	first.AddRow(named("a", "A"))
	first.SetAvailable(true)
	p.Add(first)
	p.SetArrangement(glance.Lines)
	assert.Equal(t, glance.Lines, p.Arrangement())

	late := glance.NewCard("Late")
	late.AddRow(named("b", "B"))
	late.SetAvailable(true)
	p.Add(late)

	require.True(t, shown(first.Object(), first.Rows()[0].Object()), "the card is not drawn, so its heading proves nothing")
	for _, c := range []*glance.Card{first, late} {
		header := textOf(c.Object(), map[*glance.Card]string{first: "First", late: "Late"}[c])
		require.NotNil(t, header)
		assert.False(t, shown(c.Object(), header), "a card's heading is drawn in Lines")
	}

	p.SetArrangement(glance.Stack)
	assert.True(t, shown(late.Object(), textOf(late.Object(), "Late")), "the heading did not come back")
}

// Spec 057 R7. Switching while the panel is in a window resizes it once, at
// the switch: shorter in Lines, where the heading and the plot are gone, and
// back to its height on leaving.
func TestSwitchingToLinesResizesTheWindowOnce(t *testing.T) {
	test.NewTempApp(t)
	p := glance.NewPanel(0)
	c := glance.NewCard("Cooler")
	c.AddRow(named("cpu", "CPU"), named("gpu", "GPU"))
	c.AddObject(glance.NewSparkline(10))
	c.SetAvailable(true)
	p.Add(c)
	w := test.NewTempWindow(t, nil)
	p.Attach(w)
	stacked := w.Canvas().Size().Height

	p.SetArrangement(glance.Lines)
	lines := w.Canvas().Size().Height
	assert.Less(t, lines, stacked, "the window did not shrink for Lines")

	p.SetArrangement(glance.Stack)
	assert.InDelta(t, stacked, w.Canvas().Size().Height, 1, "the window did not return to its height")
}

// barOfMeter is the bar in a meter drawn as a line: the line's second piece.
func barOfMeter(t *testing.T, m *glance.Meter) fyne.CanvasObject {
	t.Helper()
	box, ok := m.Object().(*fyne.Container)
	require.True(t, ok)
	line, ok := box.Objects[0].(*fyne.Container)
	require.True(t, ok)
	require.Len(t, line.Objects, 4)
	return line.Objects[1]
}

// barOfCell is a cell's bar: its last piece.
func barOfCell(t *testing.T, c *glance.Cell) fyne.CanvasObject {
	t.Helper()
	box, ok := c.Object().(*fyne.Container)
	require.True(t, ok)
	return box.Objects[len(box.Objects)-1]
}

// Spec 057 R8. In Lines the bars of one card start and end at the same x,
// however wide each line's name, state or figures are: a terminal's columns,
// which is where a reader looks for them.
func TestTheBarsOfACardLineUpInLines(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Mixed")
	short, long := glance.NewMeter("5h", 0), glance.NewMeter("monthly spend", 0)
	short.Set(0.1, "4 %", fd.StatusGood)
	long.Set(0.8, "$ 823.52 / $1000", fd.StatusWarn)
	long.SetTrailing("1 Oct")
	grid := glance.NewCellGrid()
	a, b := glance.NewCell("Mouse", glance.NoPercent()), glance.NewCell("A long headset name", glance.NoPercent())
	a.Set(glance.Known(glance.Percent(7), fd.StatusBad))
	a.SetBar(0.07, fd.StatusBad)
	b.Set(glance.Known(glance.Percent(100), fd.StatusGood))
	b.SetNote("Charging")
	b.SetBar(1, fd.StatusGood)
	grid.Add(a, b)
	c.Add(short, long, grid)
	c.SetAvailable(true)
	c.SetArrangement(glance.Lines)
	w := test.NewTempWindow(t, c.Object())
	w.Resize(fyne.NewSize(520, 400))

	ms, ml := barOfMeter(t, short), barOfMeter(t, long)
	assert.InDelta(t, at(ms).X, at(ml).X, 1, "the meters' bars start at different x")
	assert.InDelta(t, at(ms).X+ms.Size().Width, at(ml).X+ml.Size().Width, 1, "the meters' bars end at different x")

	ca, cb := barOfCell(t, a), barOfCell(t, b)
	assert.InDelta(t, at(ca).X, at(cb).X, 1, "the cells' bars start at different x")
	assert.InDelta(t, at(ca).X+ca.Size().Width, at(cb).X+cb.Size().Width, 1, "the cells' bars end at different x")
}
