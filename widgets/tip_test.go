package widgets

import (
	"image"
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
	require.Nil(t, area.tip(), "not yet: the pointer has only just arrived")

	require.Eventually(t, func() bool { return area.tip() != nil }, 2*time.Second, 20*time.Millisecond, "the tip never appeared")

	require.Contains(t, fynetest.Texts(area.tip()),
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
	require.Nil(t, area.tip(), "the pointer left before the tip was due")
}

// Leaving after it is up takes it down.
func TestLeavingTakesTheTipDown(t *testing.T) {
	test.NewApp()
	area, win := tipOnScreen(t, "shown then gone")
	defer win.Close()

	area.MouseIn(&desktop.MouseEvent{})
	require.Eventually(t, func() bool { return area.tip() != nil }, 2*time.Second, 20*time.Millisecond)

	area.MouseOut()
	require.Nil(t, area.tip())
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
	// A tip layer over the content, which is what a window on the shell has and
	// what keeps a tip out of an overlay. tipWithoutALayer covers the other
	// path deliberately.
	win := test.NewWindow(container.NewStack(container.NewStack(btn, area), NewTipLayer()))
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

/*
A long tip wraps inside its own background.

The first version measured the label before it had a width, so it got the size
of the text on one line, then clamped the width and kept that height: the tip
was drawn one line tall with the text running out of it, off the side of the
window. Asserting the width alone would not have caught that -- the width was
right and the height was the lie -- so this asserts both.
*/
func TestALongTipWrapsRatherThanRunningOffTheWindow(t *testing.T) {
	test.NewApp()
	long := "Hands you a rod and bait if you have none, and tops any bait stack back up " +
		"as you fish. Your own gear is left alone. Water under 300 tiles cuts fishing " +
		"power, so fish in a lake rather than a puddle."
	area, win := tipOnScreen(t, long)
	defer win.Close()

	area.MouseIn(&desktop.MouseEvent{})
	require.Eventually(t, func() bool { return area.tip() != nil }, 2*time.Second, 20*time.Millisecond)

	size := area.tip().Size()
	require.LessOrEqual(t, size.Width, tipMaxWidth, "a tip is never wider than its limit")

	oneLine := widget.NewLabel("x").MinSize().Height
	require.Greaterf(t, size.Height, oneLine*2,
		"a %d-character tip at %g wide cannot be one line tall", len(long), size.Width)
}

// The limit is the window when the window is narrower than the limit: a tip
// wider than the canvas is one that cannot be read whatever it says.
func TestATipIsNoWiderThanTheWindow(t *testing.T) {
	test.NewApp()
	narrow := fyne.NewSize(240, 200)
	text := "a tip far longer than two hundred and forty pixels of window"
	label := wrappedLabel(text)
	body := container.NewPadded(label)

	got := tipSize(text, label, body, narrow)
	require.LessOrEqual(t, got.Width, narrow.Width, "wider than the window it is drawn in")
}

// wrappedLabel is the label a tip is made of.
func wrappedLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	return l
}

/*
A tip must survive being rendered to an image, not only to a window.

The catcher draws a transparent rectangle, and the first version gave it a nil
colour. Nil is invisible under the GL painter and a nil dereference under the
software one, so it worked in the window and segfaulted anything that rendered
it to an image -- a screenshot, or this test. The difference never shows up
while clicking around the app.
*/
func TestATipRendersToAnImage(t *testing.T) {
	test.NewApp()
	area, win := tipOnScreen(t, "a note long enough to wrap onto a second line in the box")
	defer win.Close()

	area.MouseIn(&desktop.MouseEvent{})
	require.Eventually(t, func() bool { return area.tip() != nil }, 2*time.Second, 20*time.Millisecond)

	require.NotPanics(t, func() { _ = softwareRender(win.Canvas()) },
		"the software painter must be able to draw every part of a tip")
}

// softwareRender draws a canvas the way a screenshot or an image test would.
func softwareRender(c fyne.Canvas) image.Image { return c.Capture() }

/*
A control still takes its click while its own tip is showing.

This is the bug the tip layer exists for. The tip used to be a popup, and a
popup is an overlay: Fyne routes pointer events to the top overlay only, so
while one was up the walk that finds what was clicked never reached the window's
content. A non-modal popup's overlay is itself tappable and dismisses on a tap,
so the first click on a button took the tip down and went nowhere, and the
second one pressed the button. Resting the pointer on a button for half a second
and then clicking it is the ordinary way to use a button, so this happened
constantly.
*/
func TestAControlStillTakesItsClickWhileItsTipIsShowing(t *testing.T) {
	test.NewApp()
	taps := 0
	btn := widget.NewButton("Go", func() { taps++ })
	area := &tipArea{text: "what Go does", under: btn}
	area.ExtendBaseWidget(area)
	win := test.NewWindow(container.NewStack(container.NewStack(btn, area), NewTipLayer()))
	defer win.Close()
	win.Resize(fyne.NewSize(400, 200))

	area.MouseIn(&desktop.MouseEvent{})
	require.Eventually(t, func() bool { return area.tip() != nil }, 2*time.Second, 20*time.Millisecond)

	require.Nil(t, win.Canvas().Overlays().Top(), "a tip is not an overlay")

	test.Tap(btn)
	require.Equal(t, 1, taps, "one click, one press")
}

// The tip is drawn in the layer, over the content, and taken out of it again.
func TestTheTipIsDrawnInTheLayerAndRemovedFromIt(t *testing.T) {
	test.NewApp()
	area, win := tipOnScreen(t, "in the layer")
	defer win.Close()

	layer := tipLayerIn(win.Canvas())
	require.NotNil(t, layer, "the window has one")
	require.Empty(t, layer.Objects)

	area.MouseIn(&desktop.MouseEvent{})
	require.Eventually(t, func() bool { return area.tip() != nil }, 2*time.Second, 20*time.Millisecond)
	require.Len(t, layer.Objects, 1)
	require.Contains(t, fynetest.Texts(layer), "in the layer")

	area.MouseOut()
	require.Empty(t, layer.Objects, "and it does not accumulate")
}

/*
A window with no tip layer still gets its tips, through the popup.

A program that builds its own window rather than using the shell has nowhere to
draw one, and guidance that eats the next click is still better than no guidance.
The shell provides a layer for exactly this reason.
*/
func TestAWindowWithoutALayerFallsBackToAPopup(t *testing.T) {
	test.NewApp()
	btn := widget.NewButton("Go", func() {})
	area := &tipArea{text: "no layer here", under: btn}
	area.ExtendBaseWidget(area)
	win := test.NewWindow(container.NewStack(btn, area))
	defer win.Close()
	win.Resize(fyne.NewSize(400, 200))

	require.Nil(t, tipLayerIn(win.Canvas()))
	area.MouseIn(&desktop.MouseEvent{})
	require.Eventually(t, func() bool { return area.tip() != nil }, 2*time.Second, 20*time.Millisecond)
	require.Contains(t, fynetest.Texts(area.tip()), "no layer here")

	area.MouseOut()
	require.Nil(t, area.tip())
}

/*
Canary for quirk 26: an overlay takes every pointer event in the window.

FindObjectAtPositionMatching walks the top overlay *instead of* the content when
there is one, so nothing below an overlay can be clicked -- and a non-modal
popup's overlay is tappable and dismisses itself, so it consumes that click
rather than ignoring it. If Fyne ever changes this, the tip could go back to
being a popup and this test is where that shows up.
*/
func TestFyneStillSendsEveryClickToTheTopOverlay(t *testing.T) {
	test.NewApp()
	taps := 0
	btn := widget.NewButton("Go", func() { taps++ })
	win := test.NewWindow(container.NewPadded(btn))
	defer win.Close()
	win.Resize(fyne.NewSize(400, 200))

	pop := widget.NewPopUp(widget.NewLabel("over everything"), win.Canvas())
	pop.ShowAtPosition(fyne.NewPos(300, 150))
	require.NotNil(t, win.Canvas().Overlays().Top())

	// A click where the button is, with the popup up somewhere else entirely.
	at := fyne.CurrentApp().Driver().AbsolutePositionForObject(btn).
		Add(fyne.NewPos(btn.Size().Width/2, btn.Size().Height/2))
	test.TapCanvas(win.Canvas(), at)
	require.Zero(t, taps, "the overlay took it, not the button under the pointer")
	require.Nil(t, win.Canvas().Overlays().Top(), "and it dismissed itself with it")
}
