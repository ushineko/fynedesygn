package widgets_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/widgets"
)

func TestASwatchIsTheSizeItWasGiven(t *testing.T) {
	/*
		The whole reason the type exists. A button measures itself from a
		theme lookup, a padding calculation and a RichText.MinSize for a
		label that is the empty string, and a scroller asks every widget it
		holds for its minimum on every layout.
	*/
	test.NewTempApp(t)

	s := widgets.NewSwatch(fyne.NewSize(20, 20))
	require.Equal(t, fyne.NewSize(20, 20), s.MinSize())
}

func TestASwatchIsCheaperToMeasureThanAButton(t *testing.T) {
	// Named for the finding rather than the mechanism: 14% of a resize
	// profile was buttonRenderer.MinSize, for blocks with no label.
	test.NewTempApp(t)

	swatch := testing.Benchmark(func(b *testing.B) {
		s := widgets.NewSwatch(fyne.NewSize(20, 20))
		for b.Loop() {
			_ = s.MinSize()
		}
	})
	button := testing.Benchmark(func(b *testing.B) {
		w := widget.NewButton("", func() {})
		for b.Loop() {
			_ = w.MinSize()
		}
	})

	require.Less(t, swatch.NsPerOp(), button.NsPerOp(),
		"a swatch measured slower than the button it replaces")
	t.Logf("swatch %d ns/op, button %d ns/op", swatch.NsPerOp(), button.NsPerOp())
}

func TestASwatchIsTapped(t *testing.T) {
	test.NewTempApp(t)

	tapped := 0
	s := widgets.NewSwatch(fyne.NewSize(20, 20))
	s.OnTapped = func() { tapped++ }

	test.Tap(s)
	require.Equal(t, 1, tapped)
}

func TestASwatchWithNothingToDoDoesNothing(t *testing.T) {
	// A block of colour that is only a block of colour is a legitimate thing
	// to want, and it must not be a nil call.
	test.NewTempApp(t)
	require.NotPanics(t, func() { test.Tap(widgets.NewSwatch(fyne.NewSize(20, 20))) })
}

func TestASwatchRecoloursWithoutBeingRebuilt(t *testing.T) {
	/*
		The shape a palette wants: build the grid once and recolour it. The
		alternative -- rebuilding a grid of widgets when one of them changes
		-- is what makes a scene editor slow to resize in the first place.
	*/
	test.NewTempApp(t)

	s := widgets.NewSwatch(fyne.NewSize(20, 20))
	s.Fill = color.NRGBA{R: 255, A: 255}
	win := test.NewWindow(s)
	t.Cleanup(win.Close)

	s.Fill = color.NRGBA{B: 255, A: 255}
	s.Stroke = color.NRGBA{G: 255, A: 255}
	s.StrokeWidth = 3
	require.NotPanics(t, s.Refresh)
	require.Equal(t, fyne.NewSize(20, 20), s.MinSize())
}
