package fynetest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Find returns the first object of type T under o in WalkRendered order, or
// the zero value of T (nil for the pointer types Fyne uses) when there is
// none. It looks inside widgets that hide their children (a Form's items, a
// Card's content), which is where the controls a test wants to drive usually
// are; it therefore needs a test app to be running.
func Find[T fyne.CanvasObject](o fyne.CanvasObject) T {
	return first(o, func(T) bool { return true })
}

// FindButton returns the first button under o whose text is text, or nil.
func FindButton(o fyne.CanvasObject, text string) *widget.Button {
	return first(o, func(b *widget.Button) bool { return b.Text == text })
}

// FindCheck returns the first check box under o, or nil.
func FindCheck(o fyne.CanvasObject) *widget.Check {
	return Find[*widget.Check](o)
}

// FindEntry returns the first entry under o, or nil.
func FindEntry(o fyne.CanvasObject) *widget.Entry {
	return Find[*widget.Entry](o)
}

// FindSlider returns the first slider under o, or nil.
func FindSlider(o fyne.CanvasObject) *widget.Slider {
	return Find[*widget.Slider](o)
}

// FindSelect returns the first select under o, or nil.
func FindSelect(o fyne.CanvasObject) *widget.Select {
	return Find[*widget.Select](o)
}

// FindLabel returns the first label under o whose text is exactly text, or
// nil. Tests that only need to know a fact is somewhere on screen should use
// Text instead and stay independent of which widget carries it.
func FindLabel(o fyne.CanvasObject, text string) *widget.Label {
	return first(o, func(l *widget.Label) bool { return l.Text == text })
}

// first is the shared body of the finders: the first T under o in
// WalkRendered order that satisfies want, or T's zero value.
func first[T fyne.CanvasObject](o fyne.CanvasObject, want func(T) bool) T {
	var found T
	WalkRendered(o, func(obj fyne.CanvasObject) bool {
		v, ok := obj.(T)
		if !ok || !want(v) {
			return false
		}
		found = v
		return true
	})
	return found
}
