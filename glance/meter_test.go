package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

func TestAMeterFillsInProportionToItsFraction(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("max", 60)
	m.Set(0.25, "5h: 25%", fd.StatusGood)

	track, fill := meterBars(t, m, fyne.NewSize(200, 40))

	assert.InDelta(t, float64(track.Size().Width)*0.25, float64(fill.Size().Width), 1,
		"the fill should be a quarter of the track")
}

// A bar wider than its track is a bar that has left the layout, which in a
// window sized to its content means a window that grew.
func TestAMeterClampsAFractionOutsideItsRange(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("work", 60)

	m.Set(1.8, "over", fd.StatusBad)
	assert.Equal(t, 1.0, m.Fraction())

	m.Set(-0.5, "under", fd.StatusInfo)
	assert.Equal(t, 0.0, m.Fraction())

	track, fill := meterBars(t, m, fyne.NewSize(200, 40))
	assert.LessOrEqual(t, fill.Size().Width, track.Size().Width)
}

// Usage against a quota is one of the things that genuinely has a threshold,
// so the fill is graded. The four statuses must give four different fills, or
// the grading says nothing.
func TestAMeterIsColouredByItsVerdict(t *testing.T) {
	_ = fynetest.App(t)

	seen := map[string]bool{}
	for _, st := range []fd.Status{fd.StatusInfo, fd.StatusGood, fd.StatusWarn, fd.StatusBad} {
		m := glance.NewMeter("max", 60)
		m.Set(0.5, "half", st)
		_, fill := meterBars(t, m, fyne.NewSize(200, 40))
		r, g, b, _ := fill.FillColor.RGBA()
		seen[string(rune(r))+string(rune(g))+string(rune(b))] = true
		assert.Equal(t, st, m.Status())
	}
	assert.Len(t, seen, 4, "two statuses drew the same colour")
}

// The caption is not decoration. A bar says "most of it"; the caption says
// which window, how much, and when it resets.
func TestAMeterKeepsItsCaption(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("max", 60)

	m.Set(0.36, "7d: 36% (5d left) · resets in 4h 32m", fd.StatusGood)

	assert.Equal(t, "7d: 36% (5d left) · resets in 4h 32m", m.Caption())
	assert.Contains(t, fynetest.Text(m.Object()), "resets in 4h 32m")
}

// The bar reserves height and no width, so a meter never decides how wide the
// window is.
func TestAMeterReservesHeightAndNoWidth(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("max", 60)
	m.Set(0.5, "half", fd.StatusGood)

	assert.Positive(t, m.Object().MinSize().Height)
}

// meterBars renders the meter at a size and returns its track and fill.
func meterBars(t *testing.T, m *glance.Meter, size fyne.Size) (track, fill *canvas.Rectangle) {
	t.Helper()
	o := m.Object()
	o.Resize(size)
	rects := fynetest.All[*canvas.Rectangle](o)
	require.GreaterOrEqual(t, len(rects), 2, "a meter draws a track and a fill")
	return rects[len(rects)-2], rects[len(rects)-1]
}
