package fynetest

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// Text joins every string the tree under o shows with "\n": Label, Button,
// Check and Hyperlink text, the text of every RichText segment, and any bare
// canvas.Text. It looks inside widgets the way WalkRendered does, so it needs
// a test app. Tests assert that a fact is or is not on screen without knowing
// how the screen is laid out.
func Text(o fyne.CanvasObject) string {
	return strings.Join(Texts(o), "\n")
}

// Texts is Text as a slice, one entry per text-bearing object in draw order.
func Texts(o fyne.CanvasObject) []string {
	var out []string
	texts(o, &out)
	return out
}

// texts collects into out. A widget that carries text is taken at face value
// and not opened further: a Label draws a RichText that draws a canvas.Text,
// and reporting all three would list every fact three times.
func texts(o fyne.CanvasObject, out *[]string) {
	if o == nil {
		return
	}
	switch w := o.(type) {
	case *widget.Label:
		*out = append(*out, w.Text)
	case *widget.Button:
		*out = append(*out, w.Text)
	case *widget.Check:
		*out = append(*out, w.Text)
	case *widget.Hyperlink:
		*out = append(*out, w.Text)
	case *widget.RichText:
		for _, seg := range w.Segments {
			*out = append(*out, seg.Textual())
		}
	case *canvas.Text:
		*out = append(*out, w.Text)
	default:
		for _, child := range children(o, true) {
			texts(child, out)
		}
	}
}

// ScrollableIn reports whether o, or anything it draws, implements
// fyne.Scrollable. Fyne hands a wheel event to the innermost scrollable under
// the pointer and does not pass it on, so one of these inside a document is
// one place the page stops. It uses WalkRendered, so it needs a test app.
func ScrollableIn(o fyne.CanvasObject) bool {
	return WalkRendered(o, func(obj fyne.CanvasObject) bool {
		_, ok := obj.(fyne.Scrollable)
		return ok
	})
}
