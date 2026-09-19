package fynetest

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

/*
All finds what a widget drew, not what a fresh renderer would draw.

CreateRenderer *builds* a renderer: calling it hands back a new one whose list
has no rows and whose scroller has no size, so a hand-rolled walker reads a tree
that was never on screen. This one goes through the cached renderer.
*/
func TestAllFindsWhatWasDrawnRatherThanAFreshTree(t *testing.T) {
	Sandbox(t)
	app := test.NewApp()
	t.Cleanup(app.Quit)

	names := []string{"one", "two", "three"}
	list := widget.NewList(
		func() int { return len(names) },
		func() fyne.CanvasObject { return widget.NewCheck("", nil) },
		func(i widget.ListItemID, row fyne.CanvasObject) { row.(*widget.Check).Text = names[i] },
	)
	win := test.NewWindow(container.NewPadded(list))
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(300, 300))

	got := All[*widget.Check](win.Canvas().Content())
	require.Len(t, got, len(names), "one per row, from the rows the list built")

	// A fresh renderer has none of them, which is what a walker that calls
	// CreateRenderer would have found.
	require.Empty(t, list.CreateRenderer().Objects()[0].(*container.Scroll).Content.(*fyne.Container).Objects)

	first, ok := First[*widget.Check](win.Canvas().Content())
	require.True(t, ok)
	require.Same(t, got[0], first)

	_, ok = First[*widget.Hyperlink](win.Canvas().Content())
	require.False(t, ok, "nothing of that type was drawn")
}

/*
A structural walk descends a split (spec 014).

container.Split is a widget with two exported halves, so a walk that switches
on container types stops at it -- and since Shell.VSplit, every section with a
pane under a divider is one. Found in clockwork-orange, where a section became
a split and every test looking for a button in it started finding nil, while
Tips, which descends through renderers, still saw them. The two walkers
disagreed about the same tree.
*/
func TestWalkDescendsASplit(t *testing.T) {
	App(t)
	top := widget.NewButton("above", nil)
	bottom := widget.NewButton("below", nil)
	split := container.NewVSplit(top, bottom)

	found := map[string]bool{}
	Walk(split, func(o fyne.CanvasObject) bool {
		if b, ok := o.(*widget.Button); ok {
			found[b.Text] = true
		}
		return false
	})
	require.True(t, found["above"], "the leading half")
	require.True(t, found["below"], "the trailing half")
	require.Len(t, All[*widget.Button](split), 2)
}

/*
Scrolled finds the controls that move with what they act on (spec 014).

The shape under test is the one that produced the rule: a section split between
something that scrolls and a log pane, with the controls in the scrolling half.
*/
func TestScrolledFindsControlsInsideAScroller(t *testing.T) {
	App(t)
	affixed := widget.NewButton("Start", nil)
	adrift := widget.NewButton("Stop", nil)

	// Affixed in the fixed edge, adrift in the centre that scrolls.
	section := container.NewBorder(nil, affixed, nil, nil,
		container.NewVScroll(container.NewVBox(widget.NewLabel("prose"), adrift)))

	require.Equal(t, []string{"Stop"}, ScrolledButtons(section))
	require.Len(t, Scrolled[*widget.Button](section), 1)

	// Nothing in a scroller at all is the answer for a section with none.
	require.Empty(t, ScrolledButtons(container.NewVBox(affixed, adrift)))
}

// A control nested several containers deep inside the scroller is still inside
// it, and one in the other half of a split is not.
func TestScrolledLooksThroughTheWholeTree(t *testing.T) {
	App(t)
	deep := widget.NewButton("deep", nil)
	sibling := widget.NewButton("sibling", nil)
	scrolled := container.NewVScroll(container.NewVBox(container.NewHBox(container.NewVBox(deep))))

	require.Equal(t, []string{"deep"}, ScrolledButtons(container.NewVSplit(scrolled, sibling)))
}
