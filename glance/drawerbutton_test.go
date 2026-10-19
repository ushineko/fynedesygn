package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// drawerTexts is every text drawn in an object, for asking what face they
// are set in.
func drawerTexts(o fyne.CanvasObject) []*canvas.Text {
	var out []*canvas.Text
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *canvas.Text:
			out = append(out, v)
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case *container.ThemeOverride:
			walk(v.Content)
		}
	}
	walk(o)
	return out
}

// worked is a drawer with everything that answers to a pointer on it: a
// heading, a row, a lead button and a row of buttons, one of them engaged.
func worked(t *testing.T) (*glance.Window, *glance.Drawer) {
	t.Helper()
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d := w.Panel().Drawer()
	d.SetTitle("CPU by process")
	d.SetContent(glance.NewRow("a row", glance.NoRate()))
	d.SetLeadButton(&glance.DrawerSpec{
		Label: "Back", Words: []string{"Back"}, On: func() {},
	})
	d.SetButtons(
		glance.DrawerSpec{Label: "Kill", Words: []string{"Kill"}, On: func() {}},
		glance.DrawerSpec{
			Label: "Resume", Words: []string{"Pause", "Resume"},
			On: func() {}, Active: true,
		},
	)
	d.SetOpen(true)
	return w, d
}

/*
R19, and the gap that prompted it: every text the drawer draws is set in the
theme's face, the buttons' labels included.

A text takes its face only when it is told to (themed.refit). The buttons
were built without that and were not repainted when the theme changed, so
they kept the face they were born with while every row around them followed
the panel. The assertion is over every text there is rather than over the
buttons, so the next control added to the drawer is covered by it too.
*/
func TestEveryTextInTheDrawerIsSetInTheThemesFace(t *testing.T) {
	w, d := worked(t)

	th := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 20})
	w.Panel().SetTheme(th)

	found := drawerTexts(d.Frame())
	require.NotEmpty(t, found)
	var labels int
	for _, x := range found {
		assert.Equal(t, th.Font(x.TextStyle), x.FontSource,
			"the text %q is set in another face than the theme's", x.Text)
		if x.Text == "Back" || x.Text == "Kill" || x.Text == "Resume" {
			labels++
		}
	}
	assert.Equal(t, 3, labels, "the buttons' labels were not among the texts")
}

// A button built after the theme takes it, as content does (R19, and the
// drawer's own R6).
func TestAButtonAddedAfterTheThemeTakesIt(t *testing.T) {
	w, d := worked(t)
	th := fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: 20})
	w.Panel().SetTheme(th)

	d.SetButtons(
		glance.DrawerSpec{Label: "Kill", Words: []string{"Kill"}, On: func() {}},
		glance.DrawerSpec{Label: "Later", Words: []string{"Later"}, On: func() {}},
	)
	for _, x := range drawerTexts(d.Frame()) {
		if x.Text != "Later" {
			continue
		}
		assert.Equal(t, th.Font(x.TextStyle), x.FontSource)
		return
	}
	t.Fatal("the button added after the theme was not drawn")
}

// A larger face makes a larger button: the label grows, and the pill grows
// with it rather than clipping it (R17, R18).
func TestAButtonGrowsWithTheFace(t *testing.T) {
	height := func(size float32) float32 {
		w, d := worked(t)
		w.Panel().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{TextSize: size}))
		for _, x := range drawerTexts(d.Frame()) {
			if x.Text == "Kill" {
				return x.MinSize().Height
			}
		}
		t.Fatal("no Kill button")
		return 0
	}
	assert.Greater(t, height(20), height(10), "the label did not grow with the face")
}
