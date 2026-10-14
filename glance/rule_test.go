package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// A rule is one line across the whole width of the room it is given, in the
// colour of a card's border, and takes less height than a row.
func TestARuleIsALineAcrossItsRoom(t *testing.T) {
	fynetest.App(t)
	r := glance.NewRule()
	o := r.Object()
	o.Resize(fyne.NewSize(200, o.MinSize().Height))

	lines := o.(*fyne.Container).Objects
	require.Len(t, lines, 1)
	line := lines[0].(*canvas.Rectangle)
	assert.Equal(t, fyne.NewSize(200, 1), line.Size())
	assert.Equal(t, theme.Color(theme.ColorNameSeparator), line.FillColor)
	assert.Greater(t, line.Position().Y, float32(0), "the line is against the row above it")
	assert.Positive(t, o.MinSize().Height)
	assert.Less(t, o.MinSize().Height, glance.NewRow("limit", glance.NoRate()).Object().MinSize().Height)
}
