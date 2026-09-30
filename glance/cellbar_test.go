package glance_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// Spec 048. The bar's row is reserved whether or not a bar is shown, so a
// device that gains or loses a level does not reflow the card.
func TestACellIsTheSameSizeWithABarAndWithout(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502 X PLUS", "67 %", "Discharging")

	before := c.Object().MinSize()
	c.SetBar(0.67, fd.StatusGood)
	with := c.Object().MinSize()
	c.ClearBar()
	after := c.Object().MinSize()

	assert.Equal(t, before, with, "SetBar changed the cell's size")
	assert.Equal(t, before, after, "ClearBar changed the cell's size")
}

// A new cell has no bar and a cleared one has none again: nothing is drawn in
// the reserved row.
func TestACellWithNoBarDrawsNothingInItsRow(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502 X PLUS", "67 %", "Discharging")
	size := fyne.NewSize(200, c.Object().MinSize().Height)

	assert.Empty(t, drawnRects(c, size), "a new cell drew a bar")

	c.SetBar(0.5, fd.StatusGood)
	require.Len(t, drawnRects(c, size), 2, "a bar is a track and a fill")

	c.ClearBar()
	assert.Empty(t, drawnRects(c, size), "a cleared cell still drew its bar")
}

// The bar is the cell's width, filled to its fraction and coloured by its
// status, over a track in the meter's empty colour.
func TestACellsBarIsItsFractionOfTheCellsWidthInItsStatusColour(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "Arctis Nova Pro", "40 %", "Discharging")
	c.SetBar(0.4, fd.StatusWarn)

	track, fill := cellBar(t, c, 200)

	assert.InDelta(t, 200, float64(track.Size().Width), 0.5, "the track is not the cell's width")
	assert.InDelta(t, 80, float64(fill.Size().Width), 1, "the fill is not 40 % of the cell")
	assert.Equal(t, glance.BarHeight, fill.Size().Height)
	assert.Equal(t, theme.Color(theme.ColorNameWarning), fill.FillColor)
	assert.Equal(t, theme.Color(theme.ColorNameDisabledButton), track.FillColor)
}

// A fraction outside 0..1 is clamped, as a meter's is: a bar wider than its
// track has left the layout.
func TestACellsBarIsClampedToItsTrack(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502", "100 %", "Charging")

	c.SetBar(1.8, fd.StatusGood)
	track, fill := cellBar(t, c, 200)
	assert.InDelta(t, float64(track.Size().Width), float64(fill.Size().Width), 0.5)

	c.SetBar(-0.5, fd.StatusGood)
	_, fill = cellBar(t, c, 200)
	assert.Zero(t, fill.Size().Width)
}

// A stale reading dims its bar as it dims its figure, whatever the bar's
// status; a fresh reading brings the colour back.
func TestAStaleCellsBarIsDimmed(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "Arctis", "15 %", "Discharging")
	c.SetBar(0.15, fd.StatusBad)

	rd := c.Reading()
	rd.Stale = true
	c.Set(rd)
	_, fill := cellBar(t, c, 200)
	assert.Equal(t, theme.Color(theme.ColorNameDisabled), fill.FillColor)

	rd.Stale = false
	c.Set(rd)
	_, fill = cellBar(t, c, 200)
	assert.Equal(t, theme.Color(theme.ColorNameError), fill.FillColor)
}

// SetTheme and Restyle recolour the bar: its fill holds a literal colour, so
// being told about the theme is not enough.
func TestACellsBarFollowsItsTheme(t *testing.T) {
	a := fynetest.App(t)
	c := battery(t, "G502", "67 %", "Discharging")
	c.SetBar(0.67, fd.StatusGood)

	c.SetTheme(tintedTheme{Theme: a.Settings().Theme()})
	_, fill := cellBar(t, c, 200)
	assert.Equal(t, tint, fill.FillColor, "SetTheme did not recolour the bar")

	c.SetTheme(nil)
	a.Settings().SetTheme(tintedTheme{Theme: a.Settings().Theme()})
	c.Restyle()
	_, fill = cellBar(t, c, 200)
	assert.Equal(t, tint, fill.FillColor, "Restyle did not recolour the bar")
}

// tint is the success colour of tintedTheme, which no scheme uses.
var tint = color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}

// tintedTheme is a theme whose success colour is tint.
type tintedTheme struct{ fyne.Theme }

func (t tintedTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameSuccess {
		return tint
	}
	return t.Theme.Color(n, v)
}

// cellBar lays a cell out at a width and returns its bar's track and fill.
func cellBar(t *testing.T, c *glance.Cell, width float32) (track, fill *canvas.Rectangle) {
	t.Helper()
	rects := drawnRects(c, fyne.NewSize(width, c.Object().MinSize().Height))
	require.Len(t, rects, 2, "a cell with a bar draws a track and a fill")
	return rects[0], rects[1]
}

// drawnRects lays a cell out at a size and returns the rectangles it draws,
// skipping anything hidden: a hidden bar is one nobody sees.
func drawnRects(c *glance.Cell, size fyne.Size) []*canvas.Rectangle {
	o := c.Object()
	o.Resize(size)
	var out []*canvas.Rectangle
	for _, child := range o.(*fyne.Container).Objects {
		if child.Visible() {
			out = append(out, fynetest.All[*canvas.Rectangle](child)...)
		}
	}
	return out
}
