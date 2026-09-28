package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
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
