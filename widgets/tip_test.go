package widgets

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

// A tip annotates; it does not replace. With nothing to say it gets out of the
// way entirely, so a caller can annotate conditionally without branching.
func TestWithTipAnnotatesAndEmptyTextDoesNothing(t *testing.T) {
	btn := widget.NewButton("Go", func() {})
	require.NotSame(t, btn, WithTip(btn, "what Go does"))
	require.Same(t, btn, WithTip(btn, ""), "no text, no wrapper")
}

// The tip waits, so a pointer crossing a form does not leave a trail of them.
func TestATipWaitsAndThenAppears(t *testing.T) {
	test.NewApp()
	area, win := tipOnScreen(t, "Water under 300 tiles cuts fishing power.")
	defer win.Close()

	area.MouseIn(&desktop.MouseEvent{})
	require.Nil(t, area.pop, "not yet: the pointer has only just arrived")

	require.Eventually(t, func() bool {
		area.mu.Lock()
		defer area.mu.Unlock()
		return area.pop != nil
	}, 2*time.Second, 20*time.Millisecond, "the tip never appeared")

	require.Contains(t, fynetest.Texts(area.pop.Content),
		"Water under 300 tiles cuts fishing power.")
}

// Leaving before the delay shows nothing at all.
func TestLeavingBeforeTheDelayShowsNothing(t *testing.T) {
	test.NewApp()
	area, win := tipOnScreen(t, "never seen")
	defer win.Close()

	area.MouseIn(&desktop.MouseEvent{})
	area.MouseOut()

	time.Sleep(TipDelay + 200*time.Millisecond)
	area.mu.Lock()
	defer area.mu.Unlock()
	require.Nil(t, area.pop, "the pointer left before the tip was due")
}

// Leaving after it is up takes it down.
func TestLeavingTakesTheTipDown(t *testing.T) {
	test.NewApp()
	area, win := tipOnScreen(t, "shown then gone")
	defer win.Close()

	area.MouseIn(&desktop.MouseEvent{})
	require.Eventually(t, func() bool {
		area.mu.Lock()
		defer area.mu.Unlock()
		return area.pop != nil
	}, 2*time.Second, 20*time.Millisecond)

	area.MouseOut()
	area.mu.Lock()
	defer area.mu.Unlock()
	require.Nil(t, area.pop)
}

/*
The catcher sits over the control, so the control must still work.

This is the whole design: the tip takes the hover and nothing else. A tap that
stopped reaching the button underneath would be a tooltip that broke every
control it annotated.
*/
func TestAWrappedControlStillWorks(t *testing.T) {
	test.NewApp()
	tapped := 0
	btn := widget.NewButton("Go", func() { tapped++ })
	wrapped := WithTip(btn, "what Go does")

	win := test.NewWindow(wrapped)
	defer win.Close()
	win.Resize(fyne.NewSize(300, 120))

	test.Tap(btn)
	require.Equal(t, 1, tapped, "the tap must reach the control under the tip")
}

// A control that highlights under the pointer still does: only the hover
// routing was taken from it, not its behaviour.
func TestAWrappedHoverableStillHears(t *testing.T) {
	test.NewApp()
	under := &hoverCounter{}
	under.ExtendBaseWidget(under)
	area := &tipArea{text: "note", under: under}
	area.ExtendBaseWidget(area)

	area.MouseIn(&desktop.MouseEvent{})
	area.MouseMoved(&desktop.MouseEvent{})
	area.MouseOut()

	require.Equal(t, 1, under.in)
	require.Equal(t, 1, under.moved)
	require.Equal(t, 1, under.out)
}

// A tip near an edge is pulled back rather than drawn off the canvas, where it
// would be a note nobody can read.
func TestATipIsClampedToTheCanvas(t *testing.T) {
	canvasSize := fyne.NewSize(1000, 800)
	size := fyne.NewSize(200, 60)

	at := clampToCanvas(fyne.NewPos(950, 780), size, canvasSize)
	require.Equal(t, float32(800), at.X, "pulled back from the right edge")
	require.Equal(t, float32(740), at.Y, "pulled up from the bottom edge")

	at = clampToCanvas(fyne.NewPos(100, 100), size, canvasSize)
	require.Equal(t, fyne.NewPos(100, 100), at, "well inside, left alone")

	at = clampToCanvas(fyne.NewPos(-30, -30), size, canvasSize)
	require.Equal(t, fyne.NewPos(0, 0), at, "never off the top or left either")
}

/*
The canary for the rule the whole design rests on (fyne quirk).

Fyne finds the object for a pointer event by keeping the LAST match as it walks
the visible tree, so an object stacked over another wins the hover. If an
upstream change made it the first match instead, a tip would stop appearing --
and it would do so silently, because nothing else in this package would fail.
This names the behaviour so that shows up as one failing test.
*/
func TestFyneStillGivesAPointerEventToTheLastMatch(t *testing.T) {
	test.NewApp()
	under := &hoverCounter{}
	under.ExtendBaseWidget(under)
	over := &hoverCounter{}
	over.ExtendBaseWidget(over)

	win := test.NewWindow(container.NewStack(under, over))
	defer win.Close()
	win.Resize(fyne.NewSize(200, 100))

	test.MoveMouse(win.Canvas(), fyne.NewPos(100, 50))
	require.Equal(t, 1, over.in, "the object stacked on top must take the hover")
	require.Zero(t, under.in, "the one underneath must not")
}

// tipOnScreen builds a tip attached to a window, so the driver can find a
// canvas for it.
func tipOnScreen(t *testing.T, text string) (*tipArea, fyne.Window) {
	t.Helper()
	btn := widget.NewButton("Go", func() {})
	area := &tipArea{text: text, under: btn}
	area.ExtendBaseWidget(area)
	win := test.NewWindow(container.NewStack(btn, area))
	win.Resize(fyne.NewSize(400, 200))
	return area, win
}

// hoverCounter is a minimal Hoverable that records what it was told.
type hoverCounter struct {
	widget.BaseWidget
	in, moved, out int
}

func (h *hoverCounter) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel(""))
}
func (h *hoverCounter) MouseIn(*desktop.MouseEvent)    { h.in++ }
func (h *hoverCounter) MouseMoved(*desktop.MouseEvent) { h.moved++ }
func (h *hoverCounter) MouseOut()                      { h.out++ }
