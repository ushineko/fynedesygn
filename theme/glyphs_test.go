package theme_test

import (
	"testing"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// R1. The answer is a glyph the face has, and the ASCII fallback is never
// wrong: whatever the face, something comes back that can be drawn.
func TestFirstDrawableAnswersWithAGlyphTheFaceHas(t *testing.T) {
	th := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{})
	for _, candidates := range [][]string{fdtheme.BackGlyphs, fdtheme.ForwardGlyphs} {
		got := fdtheme.FirstDrawable(th, fyne.TextStyle{}, candidates...)
		assert.Contains(t, candidates, got)
	}
}

// R2. A candidate the face cannot draw is passed over for one it can.
func TestFirstDrawablePassesOverWhatTheFaceLacks(t *testing.T) {
	th := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{})
	// U+E000 is a private-use code point: no ordinary face has it.
	got := fdtheme.FirstDrawable(th, fyne.TextStyle{}, "", "<")
	assert.Equal(t, "<", got, "a private-use character was reported as drawable")
}

// R3. The last candidate is the caller's fallback: it comes back when none
// can be drawn, when there is no theme, and when the face will not parse.
func TestFirstDrawableFallsBackToTheLast(t *testing.T) {
	th := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{})
	assert.Equal(t, "", fdtheme.FirstDrawable(th, fyne.TextStyle{}, "", ""))
	assert.Equal(t, "<", fdtheme.FirstDrawable(nil, fyne.TextStyle{}, "◂", "<"))
	assert.Equal(t, "<", fdtheme.FirstDrawable(unparsable{}, fyne.TextStyle{}, "◂", "<"))
	assert.Equal(t, "", fdtheme.FirstDrawable(th, fyne.TextStyle{}))
}

// unparsable is a theme whose face is not a font.
type unparsable struct{ fyne.Theme }

func (unparsable) Font(fyne.TextStyle) fyne.Resource {
	return fyne.NewStaticResource("not-a-font", []byte("nope"))
}

func (unparsable) Size(n fyne.ThemeSizeName) float32 { return fynetheme.DefaultTheme().Size(n) }
