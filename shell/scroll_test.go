package shell

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

// affixed is a section shaped the way the design system recommends for one
// with fixed controls: a strip at the bottom and a scroller of its own above.
func affixed(title string, got **container.Scroll) *FuncSection {
	return NewSection(title, nil, func(s *Shell) fyne.CanvasObject {
		body := container.NewVBox()
		for range 200 {
			body.Add(widget.NewLabel("row"))
		}
		*got = s.VScroll("page/"+title, body)
		return container.NewBorder(nil, widget.NewButton("Apply", nil), nil, nil, *got)
	})
}

func scrollTo(sc *container.Scroll, y float32) {
	sc.ScrollToOffset(fyne.NewPos(0, y))
}

func TestARebuildKeepsTheSectionsOwnScrollPosition(t *testing.T) {
	var sc *container.Scroll
	s := onScreen(t, testOptions(affixed("Long", &sc), NewSection("Other", nil, blank)))
	s.Window.Resize(fyne.NewSize(800, 600))
	s.Select("Long")
	scrollTo(sc, 500)
	require.InDelta(t, 500, sc.Offset.Y, 0.5)

	before := sc
	s.Refresh()
	require.NotSame(t, before, sc, "the section was rebuilt")
	s.Window.Content().Refresh()
	require.InDelta(t, 500, sc.Offset.Y, 0.5, "a rebuild in place keeps the page where it was")

	s.Invalidate()
	require.InDelta(t, 500, sc.Offset.Y, 0.5, "and so does an invalidate")
}

func TestNavigatingToASectionStartsItsScrollerAtTheTop(t *testing.T) {
	var sc *container.Scroll
	s := onScreen(t, testOptions(affixed("Long", &sc), NewSection("Other", nil, blank)))
	s.Window.Resize(fyne.NewSize(800, 600))
	s.Select("Long")
	scrollTo(sc, 500)

	s.Select("Other")
	s.Select("Long")
	require.Zero(t, sc.Offset.Y)
}

func TestScrollersWithDifferentNamesKeepTheirOwnPositions(t *testing.T) {
	s := onScreen(t, testOptions(NewSection("A", nil, blank)))
	long := func() fyne.CanvasObject {
		box := container.NewVBox()
		for range 200 {
			box.Add(widget.NewLabel("row"))
		}
		return box
	}
	a := s.VScroll("a", long())
	b := s.VScroll("b", long())
	a.Offset = fyne.NewPos(0, 300)
	b.Offset = fyne.NewPos(0, 40)
	require.InDelta(t, 300, s.VScroll("a", long()).Offset.Y, 0.5)
	require.InDelta(t, 40, s.VScroll("b", long()).Offset.Y, 0.5)
}
