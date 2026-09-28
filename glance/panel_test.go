package glance_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
	fdtheme "github.com/ushineko/fynedesygn/theme"
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

// A panel can carry a theme of its own, so a program with a glance window and
// a settings window can give them different text sizes.
//
// A Fyne theme is application-wide, which is why this needs a subtree
// override rather than a second SetTheme.
func TestAPanelCanHaveAThemeOfItsOwn(t *testing.T) {
	_ = fynetest.App(t)

	p := glance.NewPanel(260)
	w := test.NewWindow(nil)
	p.Attach(w)

	small := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 8})
	p.SetTheme(small)

	assert.Equal(t, small, themeOf(t, w.Content()),
		"the panel's content is not wrapped in its own theme")
}

// Taking it away puts the panel back on the app's theme.
func TestAPanelsThemeCanBeTakenAway(t *testing.T) {
	_ = fynetest.App(t)

	p := glance.NewPanel(260)
	w := test.NewWindow(nil)
	p.Attach(w)

	p.SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 8}))
	p.SetTheme(nil)

	_, wrapped := w.Content().(*container.ThemeOverride)
	assert.False(t, wrapped, "the panel kept an override after it was taken away")
}

// A card added after the theme was set is drawn in it.
//
// Fyne's own documentation for the override says items added to the content
// afterwards keep the default theme until it is refreshed — and every consumer
// adds its cards after the window is built, so this is the ordinary path.
func TestACardAddedAfterTheThemeIsDrawnInIt(t *testing.T) {
	_ = fynetest.App(t)

	p := glance.NewPanel(260)
	w := test.NewWindow(nil)
	p.Attach(w)

	small := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 8})
	p.SetTheme(small)

	c := glance.NewCard("Added Later")
	c.AddRow(glance.NewRow("label", "--"))
	c.SetAllowed(true)
	c.SetAvailable(true)
	p.Add(c)

	// The override still wraps the content, and the card is inside it.
	assert.Equal(t, small, themeOf(t, w.Content()))
	assert.Contains(t, texts(w.Content()), "Added Later")
}

// themeOf reads the theme a content override carries.
func themeOf(t *testing.T, o fyne.CanvasObject) fyne.Theme {
	t.Helper()
	over, ok := o.(*container.ThemeOverride)
	require.True(t, ok, "the content is not a theme override")
	return over.Theme
}

// texts collects every canvas.Text in a tree, for a test that asks what is
// drawn rather than where.
func texts(o fyne.CanvasObject) []string {
	var out []string
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *canvas.Text:
			out = append(out, v.Text)
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case *container.ThemeOverride:
			walk(v.Content)
		}
	}
	walk(o)
	return out
}

/*
A panel with a theme of its own draws its cards in it, and leaves the
application's alone.

This is the whole point of the option, and the opposite of what it used to do.
The application's theme has to belong to the window that has overlays -- a
dialog, a Select's dropdown, a context menu are added to the canvas's overlay
stack rather than to any window's content, so nothing can override them. A
panel has no overlays, so the panel is the one that takes a theme of its own.

It was reported the other way round: a settings window whose font chooser
opened in the panel's face and the panel's card opacity, because the panel
owned the application's theme and a dialog is not in a window's content.
*/
func TestAPanelWithItsOwnThemeDrawsItsCardsInIt(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)

	const appSize, panelSize = 20, 9
	a.Settings().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: appSize}))

	p := glance.NewPanel(0)
	c := glance.NewCard("Cooler")
	row := glance.NewRow("CPU", "-- °C")
	c.AddRow(row)
	p.Add(c)
	c.SetAvailable(true)

	require.Equal(t, float32(appSize), firstTextSize(t, c.Object()),
		"a panel with no theme of its own follows the application")

	p.SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: panelSize}))

	assert.Equal(t, float32(panelSize), firstTextSize(t, c.Object()),
		"the card did not take the panel's theme")
	assert.Equal(t, float32(appSize),
		a.Settings().Theme().Size(theme.SizeNameText),
		"the panel changed the application's theme, which belongs to another window")
}

// A card added after the theme was set takes it too. Cards arrive after the
// window is built in every consumer there is.
func TestACardAddedAfterwardsTakesThePanelsTheme(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	a.Settings().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 20}))

	p := glance.NewPanel(0)
	p.SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 9}))

	c := glance.NewCard("Usage")
	c.AddRow(glance.NewRow("5h", "-- %"))
	p.Add(c)
	c.SetAvailable(true)

	assert.Equal(t, float32(9), firstTextSize(t, c.Object()))
}

// Everything a card is made of takes it, not only its rows: a meter and a
// grid of cells go in through AddObject and would otherwise be the two pieces
// still wearing the application's face.
func TestAMeterAndCellsTakeThePanelsThemeToo(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	a.Settings().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 20}))

	p := glance.NewPanel(0)
	c := glance.NewCard("Usage")
	meter := glance.NewMeter("max", 60)
	grid := glance.NewCellGrid()
	cell := glance.NewCell("G502", "-- %")
	grid.Add(cell)
	c.Add(meter, grid)
	p.Add(c)
	c.SetAvailable(true)

	p.SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 9}))

	assert.Equal(t, float32(9), firstTextSize(t, meter.Object()), "the meter")
	assert.Equal(t, float32(9), firstTextSize(t, cell.Object()), "the cell")
}

// nil gives the panel back to the application, which is what a panel that
// never asked for a face of its own has always done.
func TestANilThemeReturnsThePanelToTheApplication(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	a.Settings().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 20}))

	p := glance.NewPanel(0)
	c := glance.NewCard("Cooler")
	c.AddRow(glance.NewRow("CPU", "-- °C"))
	p.Add(c)
	c.SetAvailable(true)

	p.SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 9}))
	require.Equal(t, float32(9), firstTextSize(t, c.Object()))

	p.SetTheme(nil)

	assert.Equal(t, float32(20), firstTextSize(t, c.Object()))
}

// firstTextSize is the size of the first text an object draws, recovered from
// the tree rather than from the theme: the question is what reached the canvas
// objects, and the theme is only what was asked for.
func firstTextSize(t *testing.T, o fyne.CanvasObject) float32 {
	t.Helper()

	var size float32
	fynetest.WalkRendered(o, func(obj fyne.CanvasObject) bool {
		if txt, ok := obj.(*canvas.Text); ok && txt.Text != "" {
			size = txt.TextSize
			return true
		}
		return false
	})
	require.NotZero(t, size, "no text drawn")
	return size
}
