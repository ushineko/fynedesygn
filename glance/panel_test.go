package glance_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
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

// The catcher that turns a secondary tap into the context menu is stacked over
// the panel, not placed beside the cards. Beside them it took a card's worth of
// height — a visible empty band at the top of the window — and saw taps only
// inside its own strip.
func TestTheMenuCatcherAddsNoHeightToTheWindow(t *testing.T) {
	build := func(menu func() *fyne.Menu) fyne.Size {
		w := newTestWindow(t, glance.Options{Title: "Sensors", Menu: menu})
		c := glance.NewCard("AIO")
		c.AddRow(glance.NewRow("Coolant", glance.NoQuantity("°C", 2)))
		w.Panel().Add(c)
		c.SetAvailable(true)
		return w.Panel().Size()
	}

	without := build(nil)
	with := build(func() *fyne.Menu { return fyne.NewMenu("") })

	assert.Equal(t, without, with, "the context menu catcher changed the window's size")
}

// The menu is rebuilt on every tap so it can tick the current value of what it
// shows, rather than being built once and going stale.
func TestTheMenuIsBuiltFreshEveryTimeItIsOpened(t *testing.T) {
	built := 0
	w := newTestWindow(t, glance.Options{Title: "Sensors", Menu: func() *fyne.Menu {
		built++
		return fyne.NewMenu("", fyne.NewMenuItem("Quit", func() {}))
	}})
	c := glance.NewCard("AIO")
	c.AddRow(glance.NewRow("Coolant", glance.NoQuantity("°C", 2)))
	w.Panel().Add(c)
	c.SetAvailable(true)
	w.Window().Resize(fyne.NewSize(320, 200))

	w.ShowMenu(fyne.NewPos(10, 10))
	w.ShowMenu(fyne.NewPos(10, 10))

	assert.Equal(t, 2, built)
}

// A window with no menu opens nothing rather than panicking on a nil builder.
func TestAWindowWithNoMenuOpensNothing(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors"})

	assert.NotPanics(t, func() { w.ShowMenu(fyne.NewPos(10, 10)) })
}

// A card is a surface of its own, not a run of rows. Fyne cannot draw a
// translucent window (quirk 32), so the separation an alpha-capable toolkit
// would get from opacity has to come from contrast: a fill a step from the
// window's own and a hairline border. Without it three cards read as one
// column.
func TestACardDrawsItsOwnSurface(t *testing.T) {
	a := fynetest.App(t)
	c := glance.NewCard("AIO")
	c.AddRow(glance.NewRow("Coolant", glance.NoQuantity("°C", 2)))
	c.SetAvailable(true)

	faces := fynetest.All[*canvas.Rectangle](c.Object())
	require.NotEmpty(t, faces, "the card drew no surface at all")
	face := faces[0]

	assert.Equal(t, a.Settings().Theme().Color(theme.ColorNameButton, theme.VariantDark), face.FillColor,
		"the fill should be a palette token, not a literal grey")
	assert.NotNil(t, face.StrokeColor, "a card has a hairline border")
	assert.Positive(t, face.StrokeWidth)

	// A control radius is 2 in the Breeze, Oxygen and Adwaita schemes, which
	// reads as a square at card size. A card takes a surface radius.
	assert.Equal(t, glance.CardRadius, face.CornerRadius)
	assert.Greater(t, face.CornerRadius, a.Settings().Theme().Size(theme.SizeNameInputRadius),
		"a card should be rounder than an entry box")
}

// The margins are the largest single difference between a card drawn by this
// package and a section of the monitor it comes from: with the scheme's
// padding token (3 px in Breeze) a row sits against the card's own border.
// The assertion is on the card's measured width rather than on the constants,
// so it fails if the padding stops being applied as well as if it changes.
func TestACardIsInsetFromItsOwnEdge(t *testing.T) {
	c := glance.NewCard("Bandwidth")
	row := glance.NewRow("eno2", glance.NoRate())
	c.AddRow(row)

	inner := row.Object().MinSize().Width
	got := c.Object().MinSize().Width

	assert.InDelta(t, inner+2*glance.CardPadH(), got, 0.5,
		"a card's rows are not inset by the card's own margin")
	assert.Greater(t, glance.CardPadH(), 2*theme.Padding(),
		"a surface's margin has collapsed to the scheme's control padding")
}

// The gap is the only place the panel's background shows: the stack runs to
// the window's edge, the way the monitor's sections do.
func TestCardsAreSeparatedByTheCardGap(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors"})

	first, second := glance.NewCard("Bandwidth"), glance.NewCard("AIO")
	first.AddRow(glance.NewRow("eno2", glance.NoRate()))
	second.AddRow(glance.NewRow("Coolant", glance.NoQuantity("°C", 2)))
	w.Panel().Add(first, second)

	first.SetAvailable(true)
	one := w.Panel().Size().Height

	second.SetAvailable(true)
	two := w.Panel().Size().Height

	added := two - one
	assert.InDelta(t, second.Object().MinSize().Height+glance.CardGap(), added, 0.5,
		"the second card arrived without the gap that separates it from the first")
	assert.Greater(t, glance.CardGap(), 2*theme.Padding(),
		"the gap between two cards has collapsed to the scheme's control padding")
}

// A card that runs to the window's edge is the shape; a ring of window
// background around the stack reads as a frame the panel does not have.
func TestTheCardStackRunsToTheWindowsEdge(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors", MinWidth: 1})

	c := glance.NewCard("Bandwidth")
	c.AddRow(glance.NewRow("eno2", glance.NoRate()))
	w.Panel().Add(c)
	c.SetAvailable(true)

	assert.InDelta(t, c.Object().MinSize().Width, w.Panel().Size().Width, 0.5,
		"the panel padded the card away from the window's edge")
}

// The title is the section's name and not a reading. The monitor draws it in
// its inactive grey for that reason, and a title at full foreground competes
// with the numbers under it.
func TestACardsTitleIsMutedAgainstItsRows(t *testing.T) {
	c := glance.NewCard("AIO")
	c.SetAvailable(true)

	title := findText(t, c.Object(), "AIO")
	assert.Equal(t, theme.Color(theme.ColorNamePlaceHolder), title.Color,
		"the card's title is drawn at full foreground")
}

// findText walks an object tree for the canvas.Text carrying s.
func findText(t *testing.T, o fyne.CanvasObject, s string) *canvas.Text {
	t.Helper()
	var found *canvas.Text
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *canvas.Text:
			if v.Text == s {
				found = v
			}
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		}
	}
	walk(o)
	require.NotNil(t, found, "no text %q in the tree", s)
	return found
}

// Translucency is a request, and the answer is not known until the window
// exists. A program that reads it before then must get "no", because the
// alternative — assuming yes — is a theme cleared to black on a desktop that
// refused (see glance/translucent_glfw.go).
func TestATranslucentWindowIsOpaqueUntilItIsGranted(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors", Translucent: true})

	assert.False(t, w.Translucent(),
		"the window reported a transparent framebuffer it has not been given yet")
}

// A window that cannot fade itself says so rather than failing quietly. The
// test driver is not an X11 window, which is the same answer a Wayland window
// gives: ask the compositor.
func TestAWindowThatCannotFadeItselfSaysSo(t *testing.T) {
	w := newTestWindow(t, glance.Options{Title: "Sensors"})

	err := w.SetOpacity(0.7)

	require.ErrorIs(t, err, glance.ErrOpacityNeedsCompositor,
		"a window that cannot set its own opacity reported success")
}
