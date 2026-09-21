package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/ushineko/fynedesygn/widgets"
)

/*
Roomy shows a dialog at most of the window, rather than at the size Fyne
would choose.

Fyne sizes a dialog to its content's minimum, which suits a confirmation and
suits nothing that holds a list. **A scroller's minimum size is almost
nothing**, so a dialog built around one opens at the size of its buttons: a
chooser offering eighteen keys was served through a slot showing one and a
half of them, clipped at both ends, and a file browser shows about four names.

Both of those were found by somebody using them, in two different programs, a
week apart.

**Shown first, then resized.** Before Show a dialog has no window, and Resize
asks it for its minimum size -- which in Fyne 2.8.1 dereferences that nil
window and takes the process with it (docs/fyne-quirks.md, 1). That is a crash
rather than a small dialog, and it reached somebody.

	ask := dialog.NewCustomConfirm(title, "Bind", "Cancel", body, done, win)
	dialogs.Roomy(ask, win)

Sized from the window rather than to a constant, because a dialog that is
most of a small window and most of a large one is right in both, where 760x520
is a reasonable file browser on a laptop and a postage stamp on this desk.
*/
func Roomy(d dialog.Dialog, win fyne.Window) {
	widgets.HideTips()
	d.Show()
	d.Resize(RoomySize(win, ChooserSize))
}

// RoomyFraction is how much of the window a roomy dialog takes. Most of it,
// but not so much that the window behind stops being visible -- a dialog
// that covers everything reads as a new window rather than as a question
// about this one.
const RoomyFraction = 0.85

/*
RoomySize is the size Roomy would use, with least as a floor.

Exported for a caller that builds its own dialog and wants the same shape, and
because a size is easier to test than a dialog.

A window that has not been laid out reports a zero canvas, in which case the
floor is the whole answer.
*/
func RoomySize(win fyne.Window, least fyne.Size) fyne.Size {
	if win == nil {
		return least
	}
	size := win.Canvas().Size()
	return fyne.NewSize(
		max(size.Width*RoomyFraction, least.Width),
		max(size.Height*RoomyFraction, least.Height),
	)
}
