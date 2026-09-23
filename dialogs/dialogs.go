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
	fynetheme "fyne.io/fyne/v2/theme"
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

/*
closable puts a close X in the dialog's upper right, above the body.

Fyne's dialog draws a title and whatever buttons it is given, and nothing
else: there is no close affordance in the chrome, so a dialog can only be left
by finding the right button among the others. That is fine for a confirmation,
where choosing is the point, and wrong everywhere else -- a long read-only
answer, a chooser, a question somebody opened by accident. Every window on
every desktop closes from its corner, and a dialog that does not is a dialog
people hunt around in.

Above the body rather than beside the title because the title belongs to Fyne:
its dialog offers no way to put anything in the title bar. The X therefore
sits at the top right of the content, which is the same corner a reader
reaches for.

Low importance and no label: it is an affordance, not one of the choices. The
choices are the buttons along the bottom, and an X that looked like one of
them would be a third answer to a two-answer question.
*/
func closable(body fyne.CanvasObject, hide func()) fyne.CanvasObject {
	x := widget.NewButtonWithIcon("", fynetheme.CancelIcon(), hide)
	x.Importance = widget.LowImportance
	return container.NewBorder(
		container.NewBorder(nil, nil, nil, x), nil, nil, nil, body)
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
	var d *dialog.CustomDialog
	d = dialog.NewCustomWithoutButtons(title,
		closable(container.NewVBox(body), func() { d.Hide() }), win)
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
	var d *dialog.CustomDialog
	d = dialog.NewCustomWithoutButtons(title, closable(body, func() { d.Hide() }), win)
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
	var d *dialog.ConfirmDialog
	d = dialog.NewCustomConfirm(title, confirm, "Cancel",
		closable(body, func() { d.Hide() }), func(ok bool) {
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
	var d *dialog.CustomDialog
	d = dialog.NewCustom(title, "Close", closable(body, func() { d.Hide() }), win)
	d.Resize(fyne.NewSize(w, h))
	raise(win, d)
}
