package shell

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
)

/*
A banner does not take the window's clicks.

It used to be a popup, and a popup is an overlay: Fyne routes every pointer
event to the top overlay, so for the six to twelve seconds a banner was up every
click in the window went to it, and the first one dismissed it instead of doing
what it was aimed at. The comment above Flash has always said the section behind
a banner stays usable; drawing it in a layer of the content is what makes that
true (quirk 26).
*/
func TestABannerDoesNotTakeTheWindowsClicks(t *testing.T) {
	taps := 0
	o := testOptions(NewSection("A", nil, func(*Shell) fyne.CanvasObject {
		return widget.NewButton("Go", func() { taps++ })
	}))
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	s := onScreen(t, o)
	s.Window.Resize(fyne.NewSize(900, 500))

	s.Flash("Something happened.", fd.StatusWarn)
	require.Equal(t, "Something happened.", s.FlashText())
	require.Nil(t, s.Window.Canvas().Overlays().Top(), "a banner is not an overlay")

	btn := fynetest.FindButton(s.Window.Canvas().Content(), "Go")
	require.NotNil(t, btn)
	test.Tap(btn)
	require.Equal(t, 1, taps, "the click reached the section under the banner")
	require.Equal(t, "Something happened.", s.FlashText(), "and did not dismiss it")
}

/*
The banner's own dismiss still takes its click.

It is walked last and it is tappable, so it wins the taps aimed at it while
everything else in the banner falls through to the section underneath.
*/
func TestTheBannersDismissStillWorks(t *testing.T) {
	o := testOptions(NewSection("A", nil, blank))
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	s := onScreen(t, o)
	s.Window.Resize(fyne.NewSize(900, 500))

	s.Flash("A failure that waits to be dismissed.", fd.StatusBad)
	require.NotEmpty(t, s.FlashText())

	dismiss := findDismiss(s.flashes)
	require.NotNil(t, dismiss, "the banner has a dismiss")
	test.Tap(dismiss)
	require.Empty(t, s.FlashText())
}

// findDismiss is the banner's icon-only dismiss button.
func findDismiss(o fyne.CanvasObject) *widget.Button {
	switch w := o.(type) {
	case *widget.Button:
		if w.Text == "" {
			return w
		}
	case *fyne.Container:
		for _, child := range w.Objects {
			if got := findDismiss(child); got != nil {
				return got
			}
		}
	case fyne.Widget:
		for _, child := range w.CreateRenderer().Objects() {
			if got := findDismiss(child); got != nil {
				return got
			}
		}
	}
	return nil
}
