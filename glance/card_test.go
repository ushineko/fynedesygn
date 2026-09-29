package glance_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ushineko/fynedesygn/glance"
	"github.com/ushineko/fynedesygn/widgets"
)

// AC. A card says what there is no room to draw, and stops saying it.
func TestACardCarriesANoteThatCanChange(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Peripherals")
	assert.Empty(t, c.Tip(), "a card says something before it is given anything to say")

	c.SetTip("Also connected:\n  AirPods Pro  81 %  Discharging")
	assert.Contains(t, c.Tip(), "AirPods Pro")

	c.SetTip("")
	assert.Empty(t, c.Tip(), "the note outlived the reason for it")
}

// AC. A panel draws its notes in a tip layer rather than as overlays, because
// an overlay takes every pointer event in the window while it is up (quirk
// 26) -- which would stop the context menu opening whenever a note showed.
func TestAPanelHasATipLayer(t *testing.T) {
	test.NewTempApp(t)
	p := glance.NewPanel(glance.NoMinWidth)
	w := test.NewWindow(p.Content())
	t.Cleanup(w.Close)

	assert.NotNil(t, widgets.TipLayerIn(w.Canvas()))
}
