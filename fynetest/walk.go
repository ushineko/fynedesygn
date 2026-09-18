package fynetest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// Walk visits o and everything under it depth first: *fyne.Container
// children, *container.Scroll content, and the content of every
// *container.AppTabs and *container.DocTabs item, selected or not. visit
// returns true to stop. Walk returns whether it stopped.
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
