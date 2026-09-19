package fynetest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// Walk visits o and everything under it depth first: *fyne.Container
// children, *container.Scroll content, both halves of a *container.Split, and
// the content of every *container.AppTabs and *container.DocTabs item,
// selected or not. visit returns true to stop. Walk returns whether it
// stopped.
//
// A scrolled section is a widget, not a container, so its content would
// otherwise be invisible to every test that walks a whole section; the same
// goes for the pages of a tab set. Widgets that hide their children some
// other way (Form, Card) are not opened; use WalkRendered for those.
func Walk(o fyne.CanvasObject, visit func(fyne.CanvasObject) bool) bool {
	return walk(o, visit, false)
}

// WalkRendered is Walk that also descends into widgets through
// test.WidgetRenderer(w).Objects(), so it sees what a widget draws: the
// canvas.Text inside a Label, the scroller Fyne puts around a code block. It
// needs a test app, because creating a renderer consults the current theme.
func WalkRendered(o fyne.CanvasObject, visit func(fyne.CanvasObject) bool) bool {
	return walk(o, visit, true)
}

func walk(o fyne.CanvasObject, visit func(fyne.CanvasObject) bool, rendered bool) bool {
	if o == nil {
		return false
	}
	if visit(o) {
		return true
	}
	for _, child := range children(o, rendered) {
		if walk(child, visit, rendered) {
			return true
		}
	}
	return false
}

// children is what a walker descends into under o. The structural children
// (container objects, scroll content, every tab page) come first so a hidden
// tab is still reached; with rendered set, whatever the widget's renderer
// draws follows, minus any object already listed, because a Scroll's renderer
// draws its content and a tab set's renderer draws the selected page.
func children(o fyne.CanvasObject, rendered bool) []fyne.CanvasObject {
	var out []fyne.CanvasObject
	switch c := o.(type) {
	case *fyne.Container:
		out = append(out, c.Objects...)
	case *container.Scroll:
		out = append(out, c.Content)
	case *container.Split:
		// A Split is a widget with two exported halves, so a structural walk
		// would otherwise stop at it -- and since Shell.VSplit every section
		// with a pane under a divider is one. Found in clockwork-orange, where
		// a section became a split and every test looking for a button in it
		// started finding nil.
		out = append(out, c.Leading, c.Trailing)
	case *container.AppTabs:
		out = append(out, tabContents(c.Items)...)
	case *container.DocTabs:
		out = append(out, tabContents(c.Items)...)
	}
	if !rendered {
		return out
	}
	w, ok := o.(fyne.Widget)
	if !ok {
		return out
	}
	for _, drawn := range test.WidgetRenderer(w).Objects() {
		if !contains(out, drawn) {
			out = append(out, drawn)
		}
	}
	return out
}

func tabContents(items []*container.TabItem) []fyne.CanvasObject {
	out := make([]fyne.CanvasObject, 0, len(items))
	for _, item := range items {
		out = append(out, item.Content)
	}
	return out
}

// contains is a linear search; the structural list is a handful of objects
// and every Fyne canvas object is a pointer, so identity comparison is safe.
func contains(objects []fyne.CanvasObject, o fyne.CanvasObject) bool {
	for _, have := range objects {
		if have == o {
			return true
		}
	}
	return false
}

/*
All is every object of type T that o draws, in the order they are drawn.

For a test that has to reach the widgets a list or a dialog built rather than
the ones it was given: the ticks in a pick list, the buttons in a toolbar.

Use this rather than walking with CreateRenderer. CreateRenderer *builds* a
renderer; calling it hands back a fresh one whose list has no rows and whose
scroller has no size, so a hand-rolled walker reads a tree that was never on
screen. This one goes through the cached renderer, which is the tree that was.
*/
func All[T fyne.CanvasObject](o fyne.CanvasObject) []T {
	var out []T
	WalkRendered(o, func(obj fyne.CanvasObject) bool {
		if got, ok := obj.(T); ok {
			out = append(out, got)
		}
		return false // every one of them, not the first
	})
	return out
}

// First is the first object of type T that o draws, and whether there was one.
func First[T fyne.CanvasObject](o fyne.CanvasObject) (T, bool) {
	all := All[T](o)
	if len(all) == 0 {
		var none T
		return none, false
	}
	return all[0], true
}

/*
Scrolled is every object of type T under o that a scroller encloses.

For the rule that a control which starts work is affixed: it occupies the same
place in its section however much of the section is scrolled, and what scrolls
is the material it acts on. A control inside the scroller moves with that
material, and in a section that also holds a log or a table it can be worse
than moved -- Fyne hands a wheel event to the innermost scrollable under the
pointer and does not pass it on, so the control may not be reachable at all.

Structural rather than rendered: this asks where an object sits in the tree the
program built, which is a property of the layout and not of what was drawn.
*/
func Scrolled[T fyne.CanvasObject](o fyne.CanvasObject) []T {
	var out []T
	var visit func(o fyne.CanvasObject, inside bool)
	visit = func(o fyne.CanvasObject, inside bool) {
		if o == nil {
			return
		}
		if got, ok := o.(T); ok && inside {
			out = append(out, got)
		}
		if _, isScroll := o.(*container.Scroll); isScroll {
			inside = true
		}
		for _, child := range children(o, false) {
			visit(child, inside)
		}
	}
	visit(o, false)
	return out
}

/*
ScrolledButtons is the labels of every button a scroller encloses, which is the
form the check usually takes:

	require.NotContains(t, fynetest.ScrolledButtons(section), "Download now")

A program with more than a couple of these keeps the list of what must stay
affixed beside its sections and walks it, the way it keeps the list of what
must stay reachable.
*/
func ScrolledButtons(o fyne.CanvasObject) []string {
	buttons := Scrolled[*widget.Button](o)
	out := make([]string, 0, len(buttons))
	for _, b := range buttons {
		out = append(out, b.Text)
	}
	return out
}
