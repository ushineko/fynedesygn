package glance_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
	"github.com/ushineko/fynedesygn/widgets"
)

// shownTexts are the texts a row draws, in drawing order, leaving out the ones
// it is holding hidden.
func shownTexts(o fyne.CanvasObject) []*canvas.Text {
	var out []*canvas.Text
	for _, t := range fynetest.All[*canvas.Text](o) {
		if t.Visible() {
			out = append(out, t)
		}
	}
	return out
}

// appColour resolves a colour role in the test application's theme, which is
// the one a row without a theme of its own paints in.
func appColour(n fyne.ThemeColorName) color.Color {
	s := fyne.CurrentApp().Settings()
	return s.Theme().Color(n, s.ThemeVariant())
}

// bandwidth is the reading the parts exist for: a quiet download and a busy
// upload, each graded by its own size.
func bandwidth() glance.Reading {
	return glance.Parted(
		glance.Part{Text: glance.Rate(2048), Status: fd.StatusInfo},
		glance.Part{Text: " "},
		glance.Part{Text: glance.Rate(40 * (1 << 20)), Status: fd.StatusWarn, Bold: true},
	)
}

// R2. Each part is its own colour, which is the whole of hayami's complaint:
// a 2 KiB/s download painted orange because the upload beside it was busy.
func TestAPartedValuePaintsEachPartInItsOwnColour(t *testing.T) {
	_ = fynetest.App(t)
	tint := color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}
	r := glance.NewRow("eno2", glance.NoRate()+" "+glance.NoRate())
	r.Set(glance.Parted(
		glance.Part{Text: "a", Status: fd.StatusInfo},
		glance.Part{Text: "b", Status: fd.StatusBad},
		glance.Part{Text: "c", Status: fd.StatusBad, Colour: tint},
	))

	texts := shownTexts(r.Object())
	require.Len(t, texts, 4, "the label and three parts")
	assert.Equal(t, []string{"eno2", "a", "b", "c"},
		[]string{texts[0].Text, texts[1].Text, texts[2].Text, texts[3].Text})
	assert.Equal(t, appColour(fynetheme.ColorNameForeground), texts[1].Color,
		"an ungraded part is the foreground, as an ungraded value is")
	assert.Equal(t, appColour(widgets.StatusColorName(fd.StatusBad)), texts[2].Color)
	assert.Equal(t, color.Color(tint), texts[3].Color, "a part's own colour wins over its status")
	for _, p := range texts[1:] {
		assert.True(t, p.TextStyle.Monospace, "%q has to hold its column", p.Text)
	}
}

// R2. A value that is no longer being refreshed is dimmed whatever its parts
// said, the same as a plain one.
func TestAStalePartedValueIsDimmedThroughout(t *testing.T) {
	_ = fynetest.App(t)
	r := glance.NewRow("eno2", "")
	r.Set(bandwidth().Dimmed())

	disabled := appColour(fynetheme.ColorNameDisabled)
	for _, p := range shownTexts(r.Object())[1:] {
		assert.Equal(t, disabled, p.Color, "%q kept its colour while stale", p.Text)
	}

	r.Set(bandwidth())
	assert.NotEqual(t, disabled, shownTexts(r.Object())[3].Color, "fresh again, the colour comes back")
}

// R2. Bold where the part asks and the theme has the face; nowhere where it
// does not (quirk 35, the test theme having no bold mono).
func TestABoldPartIsBoldOnlyWhereTheThemeHasTheFace(t *testing.T) {
	_ = fynetest.App(t)
	for _, th := range []fyne.Theme{fyne.CurrentApp().Settings().Theme(), fynetheme.DefaultTheme()} {
		r := glance.NewRow("eno2", "")
		r.SetTheme(th)
		r.Set(bandwidth())

		parts := shownTexts(r.Object())[1:]
		require.Len(t, parts, 3)
		hasBold := th.Font(fyne.TextStyle{Monospace: true, Bold: true}) != nil
		assert.False(t, parts[0].TextStyle.Bold, "the download did not ask for bold")
		assert.Equal(t, hasBold, parts[2].TextStyle.Bold,
			"the upload asked for bold; the theme has the face: %v", hasBold)
	}
}

// R3. Parts are a way of painting the text, not a different text, so the row
// is the width the same text unparted would be.
func TestAPartedValueIsTheWidthOfTheSameTextUnparted(t *testing.T) {
	_ = fynetest.App(t)
	rd := bandwidth()

	parted := glance.NewRow("eno2", rd.Text)
	parted.Set(rd)
	plain := glance.NewRow("eno2", rd.Text)
	plain.Set(glance.Measured(rd.Text))

	assert.Equal(t, plain.Object().MinSize(), parted.Object().MinSize())
	assert.Equal(t, rd.Text, parted.Reading().Text)
	assert.Len(t, parted.Reading().Parts, 3, "Reading hands back what it was given")
}

// R3, R4. A new reading in parts of the same widths repaints the texts it has
// and moves nothing.
func TestSettingSameWidthPartsKeepsTheRowAndItsTexts(t *testing.T) {
	_ = fynetest.App(t)
	r := glance.NewRow("eno2", bandwidth().Text)
	r.Set(bandwidth())
	o := r.Object()
	o.Resize(o.MinSize())
	size := o.MinSize()
	before := shownTexts(o)
	positions := make([]fyne.Position, len(before))
	for i, t := range before {
		positions[i] = t.Position()
	}

	r.Set(glance.Parted(
		glance.Part{Text: glance.Rate(4096), Status: fd.StatusGood},
		glance.Part{Text: " "},
		glance.Part{Text: glance.Rate(1024), Status: fd.StatusInfo},
	))

	after := shownTexts(o)
	assert.Equal(t, size, o.MinSize())
	require.Len(t, after, len(before))
	for i := range before {
		assert.Same(t, before[i], after[i], "text %d was replaced rather than reused", i)
		assert.Equal(t, positions[i], after[i].Position(), "text %d moved", i)
	}
}

// R3. Switching between parts and plain text is a repaint, not a resize.
func TestSwitchingBetweenPartsAndPlainKeepsTheRowsSize(t *testing.T) {
	_ = fynetest.App(t)
	rd := bandwidth()
	r := glance.NewRow("eno2", rd.Text)
	size := r.Object().MinSize()

	r.Set(rd)
	assert.Equal(t, size, r.Object().MinSize(), "plain to parts")

	r.Set(glance.Measured(rd.Text))
	assert.Equal(t, size, r.Object().MinSize(), "parts to plain")
	assert.Len(t, shownTexts(r.Object()), 2, "plain again draws the label and one value")

	r.Set(rd)
	assert.Equal(t, size, r.Object().MinSize(), "plain to parts again")
}

// R5. A change of face repaints and refits the parts, the same as the label.
func TestAPartedRowFollowsItsTheme(t *testing.T) {
	_ = fynetest.App(t)
	r := glance.NewRow("eno2", "")
	r.Set(bandwidth())

	face := wideFace{Theme: fynetheme.DefaultTheme()}
	r.SetTheme(face)
	o := r.Object()
	o.Resize(o.MinSize())

	parts := shownTexts(o)[1:]
	for _, p := range parts {
		assert.Equal(t, face.Font(p.TextStyle), p.FontSource, "%q is measured in the wrong face", p.Text)
		assert.GreaterOrEqual(t, p.Size().Width, p.MinSize().Width, "%q is clipped", p.Text)
	}
	assert.Equal(t, face.Color(widgets.StatusColorName(fd.StatusWarn), fyne.CurrentApp().Settings().ThemeVariant()),
		parts[2].Color)
}

// R6. A pinned label column is the same width whatever the label says, which
// is what keeps a hardware name that arrives later from widening the card.
func TestAPinnedLabelKeepsTheRowsWidthWhateverItSays(t *testing.T) {
	_ = fynetest.App(t)
	short := glance.NewRowWidth("CPU", glance.NoPercent(), 90)
	long := glance.NewRowWidth("Kraken Elite V2 Liquid Cooler", glance.NoPercent(), 90)

	size := short.Object().MinSize()
	assert.Equal(t, size, long.Object().MinSize(), "a long label widened a pinned row")

	short.SetLabel("i7-13700K and then some more")
	assert.Equal(t, size, short.Object().MinSize(), "SetLabel widened a pinned row")
	short.SetLabel("")
	assert.Equal(t, size, short.Object().MinSize(), "SetLabel narrowed a pinned row")

	// And the overhang is clipped at the pin, not drawn over the value.
	o := long.Object()
	o.Resize(o.MinSize())
	label := shownTexts(o)[0]
	assert.InDelta(t, 90, label.Size().Width, 0.01, "the label is given the pinned width to draw in")
}

// R6. The difference from widgets.FixedWidth, which NewMeter uses: that is a
// floor, and a long label still widens what holds it.
func TestAPinnedLabelIsAPinWhereAMetersLabelWidthIsAFloor(t *testing.T) {
	_ = fynetest.App(t)
	long := "Kraken Elite V2 Liquid Cooler"

	shortMeter := glance.NewMeter("CPU", 90).Object().MinSize().Width
	longMeter := glance.NewMeter(long, 90).Object().MinSize().Width
	assert.Greater(t, longMeter, shortMeter, "a meter's label width is a floor; if this fails it became a pin")

	shortRow := glance.NewRowWidth("CPU", "", 90).Object().MinSize().Width
	longRow := glance.NewRowWidth(long, "", 90).Object().MinSize().Width
	assert.Equal(t, shortRow, longRow)

	// An unpinned row is as wide as its label, as it always was.
	assert.Greater(t, glance.NewRow(long, "").Object().MinSize().Width,
		glance.NewRow("CPU", "").Object().MinSize().Width)
}

// R6. A restyle measures the label at its own width (refit) and must give it
// the pinned width back.
func TestAPinnedLabelStaysPinnedThroughARestyle(t *testing.T) {
	_ = fynetest.App(t)
	face := wideFace{Theme: fynetheme.DefaultTheme()}
	long := glance.NewRowWidth("Kraken Elite V2 Liquid Cooler", glance.NoPercent(), 90)
	short := glance.NewRowWidth("CPU", glance.NoPercent(), 90)
	long.SetTheme(face)
	short.SetTheme(face)

	// The value is measured in the new face and may change width; the label
	// column may not, so the two rows still agree.
	o := long.Object()
	o.Resize(o.MinSize())
	assert.Equal(t, short.Object().MinSize(), o.MinSize())
	assert.InDelta(t, 90, shownTexts(o)[0].Size().Width, 0.01)
}

// Quirk 25 style. A parted, pinned row in a card renders to an image with the
// software painter. The assertion is that this does not panic.
func TestAPartedPinnedRowRendersToAnImage(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Bandwidth")
	r := glance.NewRowWidth("tailscale0 and a long tail", "", 70)
	c.AddRow(r)
	c.SetAvailable(true)
	r.Set(bandwidth())

	w := test.NewWindow(c.Object())
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(320, 120))
	require.NotPanics(t, func() { _ = w.Canvas().Capture() })
}
