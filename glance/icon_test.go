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

// AC. A card with no icon has none: every existing card is unchanged.
func TestACardHasNoIconUntilItIsGivenOne(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Peripherals")
	assert.Nil(t, c.Icon())
}

// AC. The icon comes before the label, because it is what the eye finds
// first when scanning a stack of cards.
func TestTheIconComesBeforeTheLabel(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Peripherals")
	c.SetIcon(fynetheme.ComputerIcon())
	require.NotNil(t, c.Icon())

	var iconX, titleX float32 = -1, -1
	c.Object().Resize(fyne.NewSize(400, 200))
	for _, o := range test.LaidOutObjects(c.Object()) {
		switch v := o.(type) {
		case *canvas.Image:
			if v.Visible() {
				iconX = v.Position().X
			}
		case *canvas.Text:
			if v.Text == "Peripherals" {
				titleX = v.Position().X
			}
		}
	}
	require.NotEqual(t, float32(-1), iconX, "no visible icon was drawn")
	require.NotEqual(t, float32(-1), titleX, "the title was not drawn")
	assert.Less(t, iconX, titleX, "the icon was drawn after the label")
}

// AC. Setting nil takes it away again.
func TestAnIconCanBeTakenAway(t *testing.T) {
	test.NewTempApp(t)
	c := glance.NewCard("Cooler")
	c.SetIcon(fynetheme.ComputerIcon())
	c.SetIcon(nil)
	assert.Nil(t, c.Icon())
}
