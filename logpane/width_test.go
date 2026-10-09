package logpane

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/widgets"
)

/*
The pane's width must not follow its log (spec 064). The counter in its
header said "1000 line(s), 19000 older dropped" by the end of a 20,000-file
install, and the header, the pane and fynstall's fixed-size wizard window
grew with it, from 820 px to 977 px in steps.

The window plays a fixed-size window's part: whenever its content asks for
more width than it has, it grows to that width, which is what Fyne does. The
pane sits beside a column of fixed width, as on the wizard's progress page.
*/
func TestALongLineNeverWidensThePane(t *testing.T) {
	a := fynetest.App(t)
	p := New(NewModel(DefaultMaxLines))
	obj := p.Widget(Options{Title: "Output", Height: 200})
	w := a.NewWindow("log")
	t.Cleanup(w.Close)
	// Beside a column of fixed width, as the wizard's progress page puts
	// it beside the step list: a row wrapped to the pane's width then makes
	// the whole content wider than the window.
	steps := widget.NewLabel("Steps")
	w.SetContent(container.NewBorder(nil, nil, widgets.FixedWidth(steps, 220), nil, obj))
	w.Resize(fyne.NewSize(600, 300))
	before := obj.MinSize().Width

	for i := range DefaultMaxLines + 1500 { // past the limit, so lines are dropped and the counter says so
		p.Log(Info, "/home/someone/.local/share/io.example.app/lib/python3.14/site-packages/"+strings.Repeat("deeply_nested/", 6)+"module.py")
		p.Log(Info, strings.Repeat("a long line with spaces in it ", 8)+string(rune('a'+i%26)))
	}
	p.Draw()
	for range 10 {
		size := w.Canvas().Size()
		if mw := w.Content().MinSize().Width; mw > size.Width {
			size.Width = mw
		}
		w.Resize(size)
		p.Draw()
	}
	require.Contains(t, p.counter.Text, "older dropped", "the counter's longest form")
	require.Equal(t, before, obj.MinSize().Width, "the pane's minimum width follows its controls, not its log")
	require.Equal(t, float32(600), w.Canvas().Size().Width, "the window never had to grow")
}
