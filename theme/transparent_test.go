package theme_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// alpha is the background's opacity in the dark variant.
func alpha(t *testing.T, th fyne.Theme) uint32 {
	t.Helper()
	_, _, _, a := th.Color(fynetheme.ColorNameBackground, fynetheme.VariantDark).RGBA()
	return a
}

// AC. The background role is the one that goes clear, and it is the only one.
func TestOnlyTheBackgroundGoesClear(t *testing.T) {
	base := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{})
	wrapped := fdtheme.WithTransparentBackground(base)

	assert.Zero(t, alpha(t, wrapped), "the background is what the painter clears with")

	// The overlay is a popup menu's fill. A transparent one would make the
	// only interface a glance window has invisible.
	assert.Equal(t,
		base.Color(fynetheme.ColorNameOverlayBackground, fynetheme.VariantDark),
		wrapped.Color(fynetheme.ColorNameOverlayBackground, fynetheme.VariantDark),
		"the overlay background was wrapped as well")
}

// AC. A clear background survives anything that sets the app's theme.
//
// This is the case a second window hits on the way up: shell sets the
// application's theme as it is built, and without the guard that put an
// opaque background back and filled a translucent panel's gaps in --
// reported as "selecting preferences makes it solid again".
func TestAClearBackgroundSurvivesANewTheme(t *testing.T) {
	a := test.NewTempApp(t)
	a.Settings().SetTheme(fdtheme.WithTransparentBackground(
		fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{})))

	next := fdtheme.New(fdtheme.BreezeLight, fdtheme.Options{})
	kept := fdtheme.KeepTransparentBackground(a, next)

	require.NotNil(t, kept)
	assert.Zero(t, alpha(t, kept), "a second window refilled the panel's gaps")
}

// AC. An opaque background is left alone: a program with no translucent
// window never sees this happen to it.
func TestAnOpaqueBackgroundIsNotWrapped(t *testing.T) {
	a := test.NewTempApp(t)
	base := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{})
	a.Settings().SetTheme(base)

	next := fdtheme.New(fdtheme.BreezeLight, fdtheme.Options{})
	kept := fdtheme.KeepTransparentBackground(a, next)

	assert.NotZero(t, alpha(t, kept), "an opaque program was made transparent")
	assert.Equal(t, next, kept, "the theme was wrapped when it did not need to be")
}
