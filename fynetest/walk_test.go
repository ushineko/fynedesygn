package fynetest

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

/*
All finds what a widget drew, not what a fresh renderer would draw.

CreateRenderer *builds* a renderer: calling it hands back a new one whose list
has no rows and whose scroller has no size, so a hand-rolled walker reads a tree
that was never on screen. This one goes through the cached renderer.
*/
func TestAllFindsWhatWasDrawnRatherThanAFreshTree(t *testing.T) {
	Sandbox(t)
	app := test.NewApp()
	t.Cleanup(app.Quit)

	names := []string{"one", "two", "three"}
	list := widget.NewList(
		func() int { return len(names) },
		func() fyne.CanvasObject { return widget.NewCheck("", nil) },
		func(i widget.ListItemID, row fyne.CanvasObject) { row.(*widget.Check).Text = names[i] },
	)
	win := test.NewWindow(container.NewPadded(list))
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(300, 300))

	got := All[*widget.Check](win.Canvas().Content())
	require.Len(t, got, len(names), "one per row, from the rows the list built")

	// A fresh renderer has none of them, which is what a walker that calls
	// CreateRenderer would have found.
	require.Empty(t, list.CreateRenderer().Objects()[0].(*container.Scroll).Content.(*fyne.Container).Objects)

	first, ok := First[*widget.Check](win.Canvas().Content())
	require.True(t, ok)
	require.Same(t, got[0], first)

	_, ok = First[*widget.Hyperlink](win.Canvas().Content())
	require.False(t, ok, "nothing of that type was drawn")
}
