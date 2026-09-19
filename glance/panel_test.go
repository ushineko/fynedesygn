package glance_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// newTestWindow builds a window on the test driver. NewWindow falls back to an
// ordinary window when the driver is not a desktop one, which is what makes
// the panel testable without a display.
func newTestWindow(t *testing.T, o glance.Options) *glance.Window {
	t.Helper()
	a := fynetest.App(t)
	return glance.NewWindow(a, o)
}

func TestAWindowIsBuiltOnADriverWithNoSplashSupport(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors", OnTop: true})

	require.NotNil(t, w.Window())
	assert.Equal(t, "Sensors", w.Window().Title())
	assert.True(t, w.Window().FixedSize(), "a glance window is sized by its content, never by hand")
}

// The two halves of the decision. A card the user hid and a card whose source
// said nothing render identically and arrive by different paths; a single test
// covering both would pass while one of them was broken.
func TestACardIsDrawnOnlyWhenAllowedAndAvailable(t *testing.T) {
	c := glance.NewCard("AIO")

	assert.False(t, c.Drawn(), "a card with no data yet is not drawn")

	c.SetAvailable(true)
	assert.True(t, c.Drawn())

	c.SetAllowed(false)
	assert.False(t, c.Drawn(), "the user's toggle takes it down")

	c.SetAvailable(false)
	c.SetAllowed(true)
	assert.False(t, c.Drawn(), "allowing a card whose source is silent draws nothing")
}

func TestHidingACardStopsItsPoll(t *testing.T) {
	c := glance.NewCard("Bandwidth")
	var polling bool
	c.OnDrawnChanged = func(drawn bool) { polling = drawn }

	c.SetAvailable(true)
	require.True(t, polling, "a card that came on screen should start its poll")

	c.SetAllowed(false)
	assert.False(t, polling, "a hidden card has no runtime cost")
}

// Quirk 34: a fixed-size Fyne window grows to fit its content and never
// shrinks back on its own. Without an explicit Resize, hiding a card leaves an
// empty band where it was.
func TestAHiddenCardShrinksTheWindow(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors"})

	first, second := glance.NewCard("Bandwidth"), glance.NewCard("AIO")
	first.AddRow(glance.NewRow("eno2", glance.NoRate()))
	second.AddRow(glance.NewRow("Coolant", glance.NoQuantity("°C", 2)))
	w.Panel().Add(first, second)

	first.SetAvailable(true)
	second.SetAvailable(true)
	both := w.Panel().Size()

	second.SetAllowed(false)
	one := w.Panel().Size()

	assert.Less(t, one.Height, both.Height,
		"the window kept the height of a card it is no longer drawing")
}

func TestTheWindowNeverGoesNarrowerThanItsFloor(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors", MinWidth: 260})

	c := glance.NewCard("X")
	c.AddRow(glance.NewRow("a", "1"))
	w.Panel().Add(c)
	c.SetAvailable(true)

	assert.GreaterOrEqual(t, w.Panel().Size().Width, float32(260))
}

// A value arriving must not resize the window. This is the same rule the
// formatter tests pin, asserted through the widget that uses it.
func TestAValueChangingMagnitudeDoesNotResizeTheWindow(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors"})

	row := glance.NewRow("eno2", glance.NoRate())
	c := glance.NewCard("Bandwidth")
	c.AddRow(row)
	w.Panel().Add(c)
	c.SetAvailable(true)

	row.Set(glance.Measured(glance.Rate(900)))
	small := w.Panel().Size()

	row.Set(glance.Measured(glance.Rate(3.9 * 1024 * 1024 * 1024)))
	large := w.Panel().Size()

	assert.Equal(t, small, large, "the panel resized because a number changed magnitude")
}

// Stale is not blank. The reader's question is whether the last value is still
// true, and a dim number answers it without moving anything.
func TestAStaleCardKeepsItsValuesAndMarksItsHeader(t *testing.T) {
	c := glance.NewCard("AIO")
	row := glance.NewRow("Coolant", glance.NoQuantity("°C", 2))
	c.AddRow(row)
	c.SetAvailable(true)
	row.Set(glance.Known(glance.Quantity(46.6, "°C", 2), fd.StatusGood))

	c.SetStale(true)

	assert.Contains(t, row.Reading().Text, "46.6", "a stale card must not blank its last reading")
	assert.True(t, row.Reading().Stale)
	assert.True(t, c.Drawn(), "a source that has gone leaves the card up")
	assert.True(t, c.Stale())
	assert.True(t, markerShown(c), "a gone source should mark the header")

	c.SetStale(false)
	assert.False(t, row.Reading().Stale)
	assert.False(t, markerShown(c), "recovery should clear the marker")
}

// markerShown reports whether the card's "(unavailable)" marker is drawn.
// fynetest.Text reports the text of hidden objects too, so it cannot answer a
// question about visibility.
func markerShown(c *glance.Card) bool {
	for _, t := range fynetest.All[*canvas.Text](c.Object()) {
		if t.Text == glance.GoneMarker {
			return t.Visible()
		}
	}
	return false
}

func TestARowThatLosesItsMetricHidesItselfAndLeavesTheCardUp(t *testing.T) {
	c := glance.NewCard("AIO")
	cpu := glance.NewRow("CPU", glance.NoQuantity("°C", 2))
	coolant := glance.NewRow("Coolant", glance.NoQuantity("°C", 2))
	c.AddRow(cpu, coolant)
	c.SetAvailable(true)

	cpu.SetShown(false)

	assert.False(t, cpu.Shown())
	assert.True(t, c.Drawn(), "one missing metric must not take the card down")
}

// A missing reading is not a failure reading. The monitor states it as
// "absence of evidence is not a stopped pump".
func TestAMissingReadingIsNotABadStatus(t *testing.T) {
	assert.Equal(t, fd.StatusInfo, glance.Missing(glance.NoRate()).Status)
	assert.Equal(t, glance.RateWidth, glance.Width(glance.Missing(glance.NoRate()).Text))
}

func TestTheCardsRenderToAnImage(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors"})

	c := glance.NewCard("AIO")
	row := glance.NewRow("Coolant", glance.NoQuantity("°C", 2))
	c.AddRow(row)
	plot := glance.NewSparkline(60)
	plot.AddSeries("coolant", color.NRGBA{R: 0x27, G: 0xae, B: 0x60, A: 0xff}, 5)
	c.AddObject(plot)
	w.Panel().Add(c)
	c.SetAvailable(true)
	row.Set(glance.Known(glance.Quantity(46.6, "°C", 2), fd.StatusGood))
	for _, v := range []float64{45.8, 46.0, 46.2, 46.6} {
		plot.Add("coolant", v)
	}

	// A window that cannot be rendered to an image is a window a screenshot
	// and half the test helpers cannot reach (quirk 25).
	assert.NotPanics(t, func() {
		test.NewWindow(w.Panel().Content()).Resize(fyne.NewSize(300, 200))
		_ = test.Canvas().Capture()
	})
}
