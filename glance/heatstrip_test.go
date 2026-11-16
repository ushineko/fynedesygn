package glance_test

import (
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// cellsOf lays the strip out at size and returns its cells, left to right.
func cellsOf(t *testing.T, h *glance.HeatStrip, size fyne.Size) []*canvas.Rectangle {
	t.Helper()
	w := test.NewWindow(h.Object())
	t.Cleanup(w.Close)
	w.Resize(size)
	h.Object().Resize(size)
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

// Spec 083 R1. One cell per value, sharing the width after the label in
// equal parts, in order.
func TestAHeatStripHasACellPerValue(t *testing.T) {
	_ = fynetest.App(t)
	h := glance.NewHeatStrip("cores", 60)
	h.Set([]float64{0, 0.5, 1, 0.25})

	cells := cellsOf(t, h, fyne.NewSize(300, 40))
	require.Len(t, cells, 4)
	for i := 1; i < len(cells); i++ {
		assert.InDelta(t, cells[0].Size().Width, cells[i].Size().Width, 0.01)
		assert.Greater(t, cells[i].Position().X, cells[i-1].Position().X+cells[i-1].Size().Width,
			"cell %d touches the one before it: no gap between", i)
	}
	last := cells[3]
	assert.LessOrEqual(t, last.Position().X+last.Size().Width, float32(300-60)+0.5)
}

// R2. A fuller cell is nearer the primary: an empty one is the track, a full
// one the primary itself.
func TestAFullerCellIsHotter(t *testing.T) {
	a := fynetest.App(t)
	h := glance.NewHeatStrip("cores", 0)
	h.Set([]float64{0, 0.5, 1, 7})

	th := a.Settings().Theme()
	v := a.Settings().ThemeVariant()
	primary := color.NRGBAModel.Convert(th.Color(theme.ColorNamePrimary, v)).(color.NRGBA)
	track := color.NRGBAModel.Convert(th.Color(theme.ColorNameDisabledButton, v)).(color.NRGBA)
	got := h.CellColours()
	assert.Equal(t, track, got[0])
	assert.Equal(t, primary, got[2])
	assert.Equal(t, primary, got[3], "a fraction over one is clamped")
	dist := func(c color.Color) float64 {
		n := color.NRGBAModel.Convert(c).(color.NRGBA)
		return math.Abs(float64(n.R)-float64(primary.R)) + math.Abs(float64(n.G)-float64(primary.G)) + math.Abs(float64(n.B)-float64(primary.B))
	}
	assert.Less(t, dist(got[1]), dist(got[0]))
	assert.Greater(t, dist(got[1]), dist(got[2]))
}

// R3. A value that is not a number is an empty cell in its place: the cells
// after it do not move.
func TestANaNCellKeepsItsPlace(t *testing.T) {
	_ = fynetest.App(t)
	h := glance.NewHeatStrip("cores", 0)
	h.Set([]float64{0.5, math.NaN(), 0.5})

	cells := cellsOf(t, h, fyne.NewSize(300, 40))
	require.Len(t, cells, 3)
	assert.Nil(t, h.CellColours()[1])
	assert.Equal(t, color.Transparent, cells[1].FillColor)
	assert.InDelta(t, cells[1].Position().X-cells[0].Position().X, cells[2].Position().X-cells[1].Position().X, 0.01)
}

// R4. The strip's height is the same for any number of cells and any values,
// and it asks for no width beyond its label: nothing it shows reflows a panel.
func TestAHeatStripNeverChangesItsSize(t *testing.T) {
	_ = fynetest.App(t)
	h := glance.NewHeatStrip("cores", 60)
	h.Set(nil)
	empty := h.Object().MinSize()
	for _, values := range [][]float64{{1}, {0, 1, 0.5}, make([]float64, 128), {math.NaN(), 2, -1}} {
		h.Set(values)
		assert.Equal(t, empty, h.Object().MinSize(), "%d values", len(values))
	}
	assert.GreaterOrEqual(t, empty.Height, glance.HeatCellHeight)
}
