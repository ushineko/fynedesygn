package markdown

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

/*
A drag measures once, not once per step (spec 015).

Fyne hands a widget a Resize for every step of a drag, from inside the event
poll, and measuring this document means rendering every block to ask its
height. On the author's machine that was 32% of all CPU during a drag of
clockwork-orange's About section, and the event queue cannot drain while it
runs, so the window moved in bursts.
*/
func TestADragMeasuresOnceNotOncePerStep(t *testing.T) {
	fynetest.App(t)
	p := New("# One\n\nA paragraph long enough to wrap when the pane narrows.\n\n## Two\n\nAnd another one.\n", Options{SettleResize: time.Hour})
	w := test.NewWindow(container.NewScroll(p))
	defer w.Close()
	w.Resize(fyne.NewSize(900, 600))

	// The container governs the pane's width, not the window's size, so the
	// baseline is whatever the first layout settled on.
	base := p.MeasuredWidth()
	require.Positive(t, base, "the first layout measures at once: a document with no heights has no shape")

	// The drag: many small steps, none of them measured yet.
	for i := range 40 {
		p.Resize(fyne.NewSize(base-float32(i)-1, 600))
	}
	require.InDelta(t, base, p.MeasuredWidth(), 0.01,
		"the heights are still the ones from before the drag")

	// Letting go settles it, once, at the width it ended on.
	require.True(t, p.Settle())
	require.InDelta(t, base-40, p.MeasuredWidth(), 0.01)
	require.False(t, p.Settle(), "nothing left to do")
}

// Detaching cancels a pending measure, which would otherwise fire into a pane
// the section has already replaced.
func TestDetachCancelsAPendingMeasure(t *testing.T) {
	fynetest.App(t)
	p := New("# One\n\nSomething to measure.\n", Options{SettleResize: time.Hour})
	w := test.NewWindow(container.NewScroll(p))
	defer w.Close()
	w.Resize(fyne.NewSize(900, 600))
	base := p.MeasuredWidth()
	p.Resize(fyne.NewSize(base-8, 600)) // a drag step, so it waits
	p.Detach()
	require.False(t, p.Settle(), "the pending measure went with the pane")
	require.InDelta(t, base, p.MeasuredWidth(), 0.01)
}

/*
A jump is measured at once; only a drag waits (spec 015).

A section being shown or a window being maximised arrives as one large change,
and delaying it would show the document at the wrong heights for as long as it
took to settle -- visible, and for no gain, because it happens once.
*/
func TestALargeChangeIsMeasuredImmediately(t *testing.T) {
	fynetest.App(t)
	p := New("# One\n\nSomething to measure.\n", Options{SettleResize: time.Hour})
	w := test.NewWindow(container.NewScroll(p))
	defer w.Close()
	w.Resize(fyne.NewSize(900, 600))

	p.Resize(fyne.NewSize(400, 600)) // more than a quarter narrower
	require.InDelta(t, 400, p.MeasuredWidth(), 0.01, "measured without waiting")
	require.False(t, p.Settle(), "nothing was left pending")
}

// The same width twice is not a resize, however it arrives.
func TestMeasuringIsSkippedWhenTheWidthIsUnchanged(t *testing.T) {
	fynetest.App(t)
	p := New("# One\n\nSomething to measure.\n", Options{SettleResize: time.Hour})
	w := test.NewWindow(container.NewScroll(p))
	defer w.Close()
	w.Resize(fyne.NewSize(900, 600))

	p.Resize(fyne.NewSize(p.MeasuredWidth(), 600))
	require.False(t, p.Settle())
}
