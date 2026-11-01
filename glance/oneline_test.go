package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// fourCells is a grid of four batteries, and one cell's minimum width.
func fourCells(t *testing.T) (*glance.CellGrid, float32) {
	t.Helper()
	g := glance.NewCellGrid()
	for _, n := range []string{"A", "B", "C", "D"} {
		g.Add(battery(t, n, "67 %", "Discharging"))
	}
	return g, g.Object().MinSize().Width
}

// Spec 075 R2. On one line, four cells are four across in a width that fits
// two, and the grid asks for the whole line: the width a glance window grows
// to.
func TestAGridOnOneLineAsksForTheWholeLine(t *testing.T) {
	_ = fynetest.App(t)
	g, cell := fourCells(t)
	o := g.Object()
	o.Resize(fyne.NewSize(cell*2+glance.CellGridGap(), o.MinSize().Height))
	require.Equal(t, 2, linesIn(g), "the grid should wrap before it is put on one line")
	oneHigh := g.Cells()[0].Object().MinSize().Height

	g.SetOneLine(true)
	o.Resize(o.MinSize())

	assert.True(t, g.OneLine())
	assert.Equal(t, 1, linesIn(g))
	assert.InDelta(t, cell*4+3*glance.CellGridGap(), o.MinSize().Width, 0.5)
	assert.InDelta(t, oneHigh, o.MinSize().Height, 0.5)
	for i := 1; i < 4; i++ {
		assert.Greater(t, g.Cells()[i].Object().Position().X, g.Cells()[i-1].Object().Position().X,
			"the cells are not left to right")
	}
}

// Spec 075 R3. A line wider than it needs is shared equally, the last cell
// ending at the grid's edge.
func TestAGridOnOneLineFillsAWiderWidth(t *testing.T) {
	_ = fynetest.App(t)
	g, _ := fourCells(t)
	g.SetOneLine(true)
	o := g.Object()
	wide := o.MinSize().Width + 200
	o.Resize(fyne.NewSize(wide, o.MinSize().Height))

	widths := map[float32]bool{}
	for _, c := range g.Cells() {
		widths[c.Object().Size().Width] = true
	}
	assert.Len(t, widths, 1, "the cells are not one width: %v", widths)
	last := g.Cells()[3].Object()
	assert.InDelta(t, wide, last.Position().X+last.Size().Width, 0.5)
}

// Spec 075 R3. A hidden cell takes no room on the line, so the grid asks for
// one cell less.
func TestAHiddenCellNarrowsALine(t *testing.T) {
	_ = fynetest.App(t)
	g, cell := fourCells(t)
	g.SetOneLine(true)

	g.Cells()[2].SetShown(false)

	assert.InDelta(t, cell*3+2*glance.CellGridGap(), g.Object().MinSize().Width, 0.5)
}

// Spec 075 R4. Lines is one cell per line whether or not the grid was asked
// for one line, and leaving Lines brings the one line back.
func TestLinesWinsOverOneLine(t *testing.T) {
	_ = fynetest.App(t)
	p := glance.NewPanel(0)
	c := glance.NewCard("Peripherals")
	g, _ := fourCells(t)
	g.SetOneLine(true)
	c.Add(g)
	c.SetAvailable(true)
	p.Add(c)

	p.SetArrangement(glance.Lines)
	g.Object().Resize(g.Object().MinSize())
	assert.Equal(t, 4, linesIn(g), "Lines should draw one cell per line")

	p.SetArrangement(glance.Stack)
	g.Object().Resize(g.Object().MinSize())
	assert.Equal(t, 1, linesIn(g), "leaving Lines should bring the one line back")
}

// Spec 075 R2, R3, R5. In a window: the window grows to the line on its own,
// and shrinks back once the consumer asks after a cell is hidden.
func TestAWindowGrowsToALineAndShrinksBack(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Peripherals", MinWidth: 260})
	c := glance.NewCard("Peripherals")
	g, _ := fourCells(t)
	c.Add(g)
	w.Panel().Add(c)
	c.SetAvailable(true)
	wrapped := w.Panel().Size()

	g.SetOneLine(true)
	w.Panel().Resize()
	line := w.Panel().Size()
	assert.Greater(t, line.Width, wrapped.Width, "the window did not grow to the line")
	assert.Less(t, line.Height, wrapped.Height, "the window kept the wrapped height")

	g.Cells()[3].SetShown(false)
	w.Panel().Resize()
	assert.Less(t, w.Panel().Size().Width, line.Width, "the window kept the width of a hidden cell")

	g.Cells()[3].SetShown(true)
	g.SetOneLine(false)
	w.Panel().Resize()
	assert.InDelta(t, wrapped.Width, w.Panel().Size().Width, 1, "the window did not return to its width")
}
