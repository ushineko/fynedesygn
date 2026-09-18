/*
Package dialogs holds the dialog shapes the consuming programs share: the
destructive confirmation whose detail says what is not touched, the prompt of
fields plus a confirm, the read-only detail view, the file and folder choosers
that do not crash, the desktop opener, and the blocking question a worker
goroutine can ask.

Ported from angou's and nmsbonker's dialogs.go (same author, MIT). Every
function takes the fyne.Window it belongs to and nothing else of the program.
*/
package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/widgets"
)

/*
raise puts a dialog up, and takes down any hover tip first.

A dialog opens because a control was used, and using a control leaves the
pointer exactly where it was: nothing tells the tip its control has been
clicked, so it sits behind the dialog until the pointer moves. Every dialog here
goes through this.
*/
func raise(_ fyne.Window, d interface{ Show() }) {
	widgets.HideTips()
	d.Show()
}

// Dialog sizes. Wide enough for a sentence of consequences without a wall of
// text; the choosers are the largest because a directory listing needs room.
var (
	DestructiveSize = fyne.NewSize(560, 320) //nolint:gochecknoglobals // sizes, not state
	WithBodySize    = fyne.NewSize(600, 360) //nolint:gochecknoglobals // sizes, not state
	PromptSize      = fyne.NewSize(620, 340) //nolint:gochecknoglobals // sizes, not state
	ChooserSize     = fyne.NewSize(760, 520) //nolint:gochecknoglobals // sizes, not state
)

/*
ConfirmDestructive is a confirmation that names what is about to happen, with
the destructive button styled as destructive and Cancel as the safe default.

The detail says what is not touched as well as what is. Every destructive path
leaves something alone that a user might reasonably fear for, and not saying
so is how a confirmation dialog becomes the thing people click through without
reading. The canonical shape is three parts: where the result goes, what
happens to what is there, what is left alone.
*/
func ConfirmDestructive(win fyne.Window, title, detail, confirm string, do func()) {
	body := widget.NewLabel(detail)
	body.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomWithoutButtons(title, container.NewVBox(body), win)
	cancel := widget.NewButton("Cancel", d.Hide)
	proceed := widget.NewButton(confirm, func() {
		d.Hide()
		do()
	})
	proceed.Importance = widget.DangerImportance
	d.SetButtons([]fyne.CanvasObject{cancel, proceed})
	d.Resize(DestructiveSize)
	raise(win, d)
}

// ConfirmWithBody is ConfirmDestructive with widgets instead of a paragraph,
// for confirmations that carry a choice ("also delete the file"). The caller
// shows it, so it can hold the body's widgets and read them in do.
func ConfirmWithBody(win fyne.Window, title string, body fyne.CanvasObject, confirm string, do func()) *dialog.CustomDialog {
	d := dialog.NewCustomWithoutButtons(title, body, win)
	cancel := widget.NewButton("Cancel", d.Hide)
	proceed := widget.NewButton(confirm, func() {
		d.Hide()
		do()
	})
	proceed.Importance = widget.DangerImportance
	d.SetButtons([]fyne.CanvasObject{cancel, proceed})
	d.Resize(WithBodySize)
	return d
}

// Prompt is the shape most input dialogs take: a heading, one or more fields,
// and a confirm that hands the values to the caller.
func Prompt(win fyne.Window, title, confirm string, body fyne.CanvasObject, do func()) {
	d := dialog.NewCustomConfirm(title, confirm, "Cancel", body, func(ok bool) {
		if ok {
			do()
		}
	}, win)
	d.Resize(PromptSize)
	raise(win, d)
}

// ShowDetail puts a long, read-only answer on screen. A dialog rather than a
// section because it is the answer to a question that was just asked, and it
// should go away when it has been read.
func ShowDetail(win fyne.Window, title string, body fyne.CanvasObject, w, h float32) {
	d := dialog.NewCustom(title, "Close", body, win)
	d.Resize(fyne.NewSize(w, h))
	raise(win, d)
}
