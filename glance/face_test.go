package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// wideFace is a theme whose font is deliberately not the application's, so a
// text measured in the wrong one comes out a different width.
type wideFace struct{ fyne.Theme }

func (w wideFace) Font(fyne.TextStyle) fyne.Resource { return fynetheme.DefaultTextBoldFont() }

// texts are the canvas.Text objects under an object, laid out.
func faceTexts(o fyne.CanvasObject) []*canvas.Text {
	var out []*canvas.Text
	for _, x := range test.LaidOutObjects(o) {
		if t, ok := x.(*canvas.Text); ok {
			out = append(out, t)
		}
	}
	return out
}

/*
AC. A panel with a face of its own measures its text in that face.

Fyne measures in the application's font unless a text carries its own source:
canvas.Text.MinSize passes FontSource, nil means the app's theme, and the
painter meanwhile draws in whatever theme covers the object. A panel with its
own face therefore reserved the width of one family and drew another, and
where the drawn one was wider the last glyph fell off -- "CPU" as "CPL".
*/
func TestAPanelsTextsCarryItsOwnFace(t *testing.T) {
	test.NewTempApp(t)
	p := glance.NewPanel(glance.NoMinWidth)
	c := glance.NewCard("Cooler")
	c.AddRow(glance.NewRow("Coolant", "40 °C"))
	c.SetAvailable(true)
	p.Add(c)

	face := wideFace{Theme: fynetheme.DefaultTheme()}
	p.SetTheme(face)
	p.Content().Resize(fyne.NewSize(400, 300))

	found := faceTexts(p.Content())
	require.NotEmpty(t, found, "the panel drew no text at all")
	for _, x := range found {
		assert.Equal(t, face.Font(x.TextStyle), x.FontSource,
			"%q measures in the application's font, not the panel's", x.Text)
	}
}

// AC. And each text is wide enough for what it says, in that face.
func TestEveryTextIsWideEnoughForItself(t *testing.T) {
	test.NewTempApp(t)
	p := glance.NewPanel(glance.NoMinWidth)
	c := glance.NewCard("Peripherals")
	c.AddRow(glance.NewRow("tailscale0", "64.9 GiB"))
	c.SetAvailable(true)
	p.Add(c)

	p.SetTheme(wideFace{Theme: fynetheme.DefaultTheme()})
	p.Content().Resize(fyne.NewSize(400, 300))

	for _, x := range faceTexts(p.Content()) {
		if x.Text == "" {
			continue
		}
		assert.GreaterOrEqual(t, x.Size().Width, x.MinSize().Width,
			"%q is drawn in %v but needs %v, so its last glyph is clipped",
			x.Text, x.Size().Width, x.MinSize().Width)
	}
}
