package fynetest

import (
	"fyne.io/fyne/v2"
)

// tipper is what a hover note looks like from outside. Declared here rather
// than imported, because the package that implements it has tests that import
// this one, and an import back would be a cycle.
type tipper interface{ Tip() string }

/*
Tips is every hover note in a tree, in the order they are found.

A tip is raised a second after the pointer stops, so its text is
nowhere in the widget tree until somebody hovers. That leaves a headless test no
way to check the one place tips are not decoration: a control drawn as an icon
alone, where the tip is the only thing that says what it is.
*/
func Tips(o fyne.CanvasObject) []string {
	var out []string
	tips(o, &out)
	return out
}

func tips(o fyne.CanvasObject, out *[]string) {
	if o == nil {
		return
	}
	if t, ok := o.(tipper); ok {
		if text := t.Tip(); text != "" {
			*out = append(*out, text)
		}
	}
	switch w := o.(type) {
	case *fyne.Container:
		for _, child := range w.Objects {
			tips(child, out)
		}
	case fyne.Widget:
		for _, child := range w.CreateRenderer().Objects() {
			tips(child, out)
		}
	}
}
