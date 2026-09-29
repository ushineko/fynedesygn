package glance

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A cell's reading is bold where the theme has the face, and is not a panic
where it does not.

Canary for quirk 35. Fyne reads the font the theme returns for a style and
does not fall back, so a mono family with no bold is a nil that the painter
dereferences. Fyne's own test theme is one of those, which is what makes this
testable at all: if this ever starts reporting Bold under the test theme,
Fyne has grown a fallback and themed.bold can go.
*/
func TestACellsReadingIsBoldOnlyWhereTheThemeHasTheFace(t *testing.T) {
	test.NewTempApp(t)
	c := NewCell("G502 X PLUS", NoPercent())

	assert.True(t, c.value.TextStyle.Monospace, "a reading has to hold its columns")

	th := fyne.CurrentApp().Settings().Theme()
	if th.Font(fyne.TextStyle{Monospace: true, Bold: true}) == nil {
		assert.False(t, c.value.TextStyle.Bold,
			"a face the theme does not have is a panic in the painter, not a lighter weight")
		return
	}
	assert.True(t, c.value.TextStyle.Bold,
		"the theme has a bold mono face and the reading did not use it")
}

// The name and the state stay regular, so the three pieces of a cell read in
// the order they matter.
func TestOnlyTheReadingIsBold(t *testing.T) {
	test.NewTempApp(t)
	c := NewCell("G502 X PLUS", NoPercent())

	assert.False(t, c.name.TextStyle.Bold, "the name competes with the reading")
	assert.False(t, c.note.TextStyle.Bold, "the state competes with the reading")
}

// A cell drawn under a theme with no bold mono face still draws. The
// assertion is that this does not panic.
func TestACellRendersUnderAThemeWithNoBoldMono(t *testing.T) {
	test.NewTempApp(t)
	c := NewCell("G502 X PLUS", NoPercent())
	c.Set(Measured("72"))

	require.NotPanics(t, func() {
		w := test.NewWindow(c.Object())
		t.Cleanup(w.Close)
		w.Resize(fyne.NewSize(300, 200))
	})
}
