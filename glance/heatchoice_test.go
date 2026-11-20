package glance_test

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// fortnight is fourteen cells, a day each.
var fortnight = []float64{0.2, 0, 0.4, 0.6, 1, 0.3, 0, 0, 0.5, 0.7, 0.9, 0.2, 0.4, 0.8}

// days are a fortnight's initials.
var days = strings.Split("FSSMTWTFSSMTWT", "")

// labelsOf are the strip's cell labels as drawn, left to right.
func labelsOf(t *testing.T, h *glance.HeatStrip) []*canvas.Text {
	t.Helper()
	var out []*canvas.Text
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if tx, ok := o.(*canvas.Text); ok && tx.Text != h.Label() {
			out = append(out, tx)
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, k := range c.Objects {
				walk(k)
			}
		}
		if wd, ok := o.(fyne.Widget); ok {
			for _, k := range test.WidgetRenderer(wd).Objects() {
				walk(k)
			}
		}
	}
	walk(h.Object())
	return out
}

// centre is the middle of an object, along x, in its parent's coordinates.
func centre(o fyne.CanvasObject) float32 { return o.Position().X + o.Size().Width/2 }

// Spec 085 R1, AC1. A label per cell, centred under it.
func TestEachCellHasItsLabelUnderIt(t *testing.T) {
	_ = fynetest.App(t)
	h := glance.NewHeatStrip("", 0)
	h.Set(fortnight)
	h.SetCellLabels(days)

	cells := cellsOf(t, h, fyne.NewSize(420, 60))
	labels := labelsOf(t, h)
	require.Len(t, cells, 14)
	require.Len(t, labels, 14)
	for i := range cells {
		assert.Equal(t, days[i], labels[i].Text)
		assert.InDelta(t, centre(cells[i]), centre(labels[i]), 0.5, "label %d is not under its cell", i)
		assert.GreaterOrEqual(t, labels[i].Position().Y, cells[i].Position().Y+glance.HeatCellHeight,
			"label %d overlaps its cell", i)
	}
}

// R1, AC2. With labels the strip is one fixed height, whatever they say and
// however many cells; nil puts it back to the cells' height.
func TestLabelsAddOneFixedLine(t *testing.T) {
	_ = fynetest.App(t)
	h := glance.NewHeatStrip("", 0)
	h.Set(fortnight)
	plain := h.Object().MinSize()
	h.SetCellLabels(days)
	labelled := h.Object().MinSize()
	assert.Greater(t, labelled.Height, plain.Height, "labels took no room")

	for _, n := range []int{1, 14, 128} {
		values := make([]float64, n)
		words := make([]string, n)
		for i := range words {
			words[i] = strings.Repeat("Wy", i%4+1)
		}
		h.Set(values)
		h.SetCellLabels(words)
		assert.Equal(t, labelled, h.Object().MinSize(), "%d cells", n)
	}
	h.SetSelected(0)
	assert.Equal(t, labelled, h.Object().MinSize(), "a bold label changed the height")

	h.SetCellLabels(nil)
	assert.Equal(t, plain, h.Object().MinSize())
}

// R2, AC3. Selecting outlines that cell and no other; -1 and an index past
// the end outline none, and nothing changes size.
func TestSelectingACellOutlinesIt(t *testing.T) {
	_ = fynetest.App(t)
	h := glance.NewHeatStrip("", 0)
	h.Set(fortnight)
	h.SetCellLabels(days)
	size := h.Object().MinSize()
	outlined := func() []int {
		var out []int
		for i, c := range cellsOf(t, h, fyne.NewSize(420, 60)) {
			// A NaN cell is outlined too, at width 1; the mark is wider.
			if c.StrokeWidth > 1 {
				out = append(out, i)
			}
		}
		return out
	}

	assert.Empty(t, outlined())
	assert.Equal(t, -1, h.Selected())

	h.SetSelected(4)
	assert.Equal(t, []int{4}, outlined())
	assert.Equal(t, 4, h.Selected())
	labels := labelsOf(t, h)
	assert.True(t, labels[4].TextStyle.Bold)
	assert.False(t, labels[3].TextStyle.Bold)
	assert.Equal(t, size, h.Object().MinSize())

	h.SetSelected(-1)
	assert.Empty(t, outlined())
	h.SetSelected(14)
	assert.Empty(t, outlined())
	assert.Equal(t, -1, h.Selected())
}

// R3, AC4. CellAt is the cell at its middle and in the gap after it, and -1
// on the label, on the labels' line and outside.
func TestCellAtFindsTheCell(t *testing.T) {
	_ = fynetest.App(t)
	h := glance.NewHeatStrip("busy", 60)
	h.Set(fortnight)
	h.SetCellLabels(days)
	cells := cellsOf(t, h, fyne.NewSize(480, 60))
	drv := fyne.CurrentApp().Driver()

	for i, c := range cells {
		at := drv.AbsolutePositionForObject(c)
		mid := at.AddXY(c.Size().Width/2, c.Size().Height/2)
		assert.Equal(t, i, h.CellAt(mid), "the middle of cell %d", i)
		gap := at.AddXY(c.Size().Width+glance.HeatCellGap/2, c.Size().Height/2)
		if i < len(cells)-1 {
			assert.Equal(t, i, h.CellAt(gap), "the gap after cell %d", i)
		}
	}
	first := drv.AbsolutePositionForObject(cells[0])
	assert.Equal(t, -1, h.CellAt(first.AddXY(-10, 4)), "the label is not a cell")
	assert.Equal(t, -1, h.CellAt(first.AddXY(4, glance.HeatCellHeight+4)), "the labels' line is not a cell")
	assert.Equal(t, -1, h.CellAt(fyne.NewPos(-1, -1)))
}

// R4, AC5. A tap on a cell in an open drawer is the cell's: the strip and its
// index reach SetCellTapped's function and not SetRowTapped's; a tap on a
// row still reaches the row's.
func TestATapOnACellIsTheCells(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "cells"})
	dr := w.Panel().Drawer()
	row := glance.NewRow("GitHub", "")
	strip := glance.NewHeatStrip("", 0)
	strip.Set(fortnight)
	strip.SetCellLabels(days)
	dr.SetContent(row, strip)
	dr.SetSide(glance.Right)
	dr.SetOpen(true)

	var gotStrip *glance.HeatStrip
	gotCell, rows := -1, 0
	dr.SetCellTapped(func(h *glance.HeatStrip, i int) { gotStrip, gotCell = h, i }, strip)
	dr.SetRowTapped(func(*glance.Row) { rows++ }, row)

	cv := w.Window().Canvas()
	cells := cellsIn(strip)
	require.Len(t, cells, 14)
	test.TapCanvas(cv, middle(cells[9]))
	assert.Same(t, strip, gotStrip)
	assert.Equal(t, 9, gotCell)
	assert.Zero(t, rows, "the tap on a cell reached the rows too")

	test.TapCanvas(cv, middle(row.Object()))
	assert.Equal(t, 1, rows)
	assert.Equal(t, 9, gotCell, "the tap on the row reached the cells")

	dr.SetCellTapped(nil)
	gotCell = -1
	test.TapCanvas(cv, middle(cells[2]))
	assert.Equal(t, -1, gotCell, "cells answered with no function set")
}

// cellsIn are a laid-out strip's cells, left to right.
func cellsIn(h *glance.HeatStrip) []*canvas.Rectangle {
	var out []*canvas.Rectangle
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if r, ok := o.(*canvas.Rectangle); ok && r.Size().Height == glance.HeatCellHeight {
			out = append(out, r)
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, k := range c.Objects {
				walk(k)
			}
		}
		if wd, ok := o.(fyne.Widget); ok {
			for _, k := range test.WidgetRenderer(wd).Objects() {
				walk(k)
			}
		}
	}
	walk(h.Object())
	return out
}
