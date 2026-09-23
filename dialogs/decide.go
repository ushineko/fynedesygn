package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

/*
Decide asks a yes-or-no question from a worker goroutine and blocks until it
is answered.

Every core call runs on a goroutine and every callback hops back to the UI
thread with fyne.Do to put its question on screen, then waits on a channel for
the answer. That is the whole trick, and it is why this must never be called
from a widget handler or anywhere else on the UI thread: it would wait for an
answer the blocked thread can no longer deliver.

A dialog closed by any path (a button, the window) answers exactly once.
*/
func Decide(win fyne.Window, title, detail, yes, no string) bool {
	answer := make(chan bool, 1)
	fyne.Do(func() { ask(win, title, detail, yes, no, answer) })
	return <-answer
}

// ask puts the question up on the UI thread and returns the dialog. sent
// makes every close path answer exactly once, including a dismissal that
// never touched a button.
func ask(win fyne.Window, title, detail, yes, no string, answer chan<- bool) *dialog.CustomDialog {
	body := widget.NewLabel(detail)
	body.Wrapping = fyne.TextWrapWord
	var d *dialog.CustomDialog
	d = dialog.NewCustomWithoutButtons(title,
		closable(container.NewVBox(body), func() { d.Hide() }), win)

	sent := false
	send := func(v bool) {
		if sent {
			return
		}
		sent = true
		answer <- v
	}
	noBtn := widget.NewButton(no, func() { d.Hide() })
	yesBtn := widget.NewButton(yes, func() {
		send(true)
		d.Hide()
	})
	yesBtn.Importance = widget.HighImportance
	d.SetButtons([]fyne.CanvasObject{noBtn, yesBtn})
	d.SetOnClosed(func() { send(false) })
	d.Resize(fyne.NewSize(500, 240))
	raise(win, d)
	return d
}
