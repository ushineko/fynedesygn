package dialogs

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

func TestARoomyDialogIsMostOfTheWindow(t *testing.T) {
	/*
		Fyne sizes a dialog to its content's minimum, and a scroller's
		minimum is almost nothing -- so a dialog built around one opens at
		the size of its buttons. Found twice in a week, in two programs: a
		file browser showing four names, and a chooser offering eighteen keys
		through a slot showing one and a half.
	*/
	test.NewTempApp(t)
	win := test.NewWindow(widget.NewLabel("behind"))
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(1600, 1000))

	got := RoomySize(win, ChooserSize)
	require.InDelta(t, 1600*RoomyFraction, got.Width, 1)
	require.InDelta(t, 1000*RoomyFraction, got.Height, 1)
}

func TestASmallWindowStillGetsAUsableDialog(t *testing.T) {
	// The floor. Most of a very small window is still too small to read a
	// directory in.
	test.NewTempApp(t)
	win := test.NewWindow(widget.NewLabel("behind"))
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(300, 200))

	got := RoomySize(win, ChooserSize)
	require.Equal(t, ChooserSize, got)
}

func TestSizingAWindowlessDialogIsTheFloor(t *testing.T) {
	// A caller with no window is a headless one, and a nil dereference is
	// what this whole helper exists to avoid.
	require.Equal(t, ChooserSize, RoomySize(nil, ChooserSize))
}
