package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// A glance window is its content by default, which is what makes it a glance
// window: nothing has to be dragged to fit.
func TestAGlanceWindowIsFixedToItsContentByDefault(t *testing.T) {
	a := fynetest.App(t)

	w := glance.NewWindow(a, glance.Options{Title: "fixed"})
	assert.True(t, w.Window().FixedSize(),
		"a glance window should be its content unless the caller says otherwise")
}

// Options.Resizable hands the size back to the window manager.
//
// What it buys is not obvious from here: a fixed-size window tells the window
// manager it will not take a resize, so the compositor's own Resize — the one
// in the window menu a frameless window still has — is greyed out. A user who
// wants their panel wider has no way to say so.
func TestAResizableGlanceWindowLetsTheWindowManagerIn(t *testing.T) {
	a := fynetest.App(t)

	w := glance.NewWindow(a, glance.Options{Title: "resizable", Resizable: true})
	assert.False(t, w.Window().FixedSize())
}

// A resizable panel does not pull the window back to its content's width.
//
// The shrink-back is quirk 34's: a fixed-size Fyne window grows to fit and
// never shrinks, so a card that hides leaves an empty band unless something
// lowers the requested size. That still has to happen — but only down to the
// width the user chose, never past it.
func TestAResizablePanelKeepsTheWidthTheUserChose(t *testing.T) {
	a := fynetest.App(t)

	w := glance.NewWindow(a, glance.Options{Title: "resizable", Resizable: true})
	c := glance.NewCard("A Card")
	w.Panel().Add(c)
	w.Window().Show()

	wide := w.Panel().Size().Width + 300
	w.Window().Resize(fyne.NewSize(wide, w.Panel().Size().Height))

	// A card hiding is what calls Resize, and is the case quirk 34 is about.
	c.SetAllowed(false)
	w.Panel().Resize()

	assert.GreaterOrEqual(t, w.Window().Canvas().Size().Width, wide,
		"the panel pulled the window back to its content and lost the user's width")
}

// Translucency is the glance window's own and does not reach the app.
//
// It used to be done by swapping the app's theme, and a Fyne theme is
// app-wide: a program with a second window — which shell.NewIn exists for —
// had that window's background turned transparent too, and a window that is
// not translucent draws a transparent background as black.
func TestTranslucencyDoesNotReachTheAppsTheme(t *testing.T) {
	a := fynetest.App(t)
	before := a.Settings().Theme()

	w := glance.NewWindow(a, glance.Options{Title: "glance", Translucent: true})
	w.Panel().SetTranslucent(true)

	assert.Same(t, before, a.Settings().Theme(),
		"the app's theme was replaced, which turns every other window transparent")
}

// A panel that is not translucent paints the scheme's background, so a glance
// window on a desktop that refused the grant looks as it always did.
func TestAnOpaquePanelPaintsTheSchemesBackground(t *testing.T) {
	a := fynetest.App(t)

	w := glance.NewWindow(a, glance.Options{Title: "glance"})
	w.Panel().SetTranslucent(false)

	assert.Equal(t, theme.Color(theme.ColorNameBackground), w.Panel().BackgroundColour())
}

// And a translucent one paints nothing at all, which is what lets the desktop
// through the space between cards.
func TestATranslucentPanelPaintsNothing(t *testing.T) {
	a := fynetest.App(t)

	w := glance.NewWindow(a, glance.Options{Title: "glance", Translucent: true})
	w.Panel().SetTranslucent(true)

	_, _, _, alpha := w.Panel().BackgroundColour().RGBA()
	assert.Zero(t, alpha)
}
